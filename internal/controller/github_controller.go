/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/go-github/v64/github"
	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	core "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// The Environment Key needs to be set for a Github deployment to occur.
// The value for this key should be the *name* of the environment you want to
// create a deployment for.
// The environment needs to be created by the user. It's possible to create one through the Github UI,
// more information about environment deployment is available on Github[1].
//
// [1]: https://docs.github.com/en/rest/deployments/environments?apiVersion=2022-11-28#about-deployment-environments
const kGithubEnvironmentKey string = "github.se.quencer.io/environment"

// The Ref Key needs to be set for a Github deployment to occur.
// The value for this key should be the valid *name* of the branch that you want to
// create a deployment for.
// This is required by Github[1] but will not have any effect on the deployment.
//
// [1]: https://docs.github.com/en/rest/deployments/deployments?apiVersion=2022-11-28#create-a-deployment
const kGithubRefKey string = "github.se.quencer.io/reference"

// The deployment key is an optional field that references an **existing** deployment. If the
// value is not present, the reconciler will create a new Deployment[1] with some hard-coded values.
// The value is the ID of the deployment. If the deployment needs to be created by the reconciler, the value
// needs to be unset and the reconciler will set it to the ID once the deployment is created.
//
// [1]: https://docs.github.com/en/rest/deployments/deployments?apiVersion=2022-11-28#create-a-deployment
const kGithubDeploymentKey string = "github.se.quencer.io/deployment"

// The Owner should be the owner of the repository where the deployment should be created.
// This value is the name of the owner, and needs to be the owner of the repository set in the annotation.
const kGithubOwnerKey string = "github.se.quencer.io/owner"

// The Repository is the name of the repository owned by the owner (set above) that is going to be used
// to deploy to the environment.
const kGithubRepositoryKey string = "github.se.quencer.io/repository"

// The finalizer is in charge of disabling the deployment on Github before the Workspace is deleted,
// this will make sure that the states on Github matches as closely as possible the state of Sequencer.
const kGithubFinalizer string = "github.se.quencer.io/finalizer"

// The Condition that will be added to the Workspace Status to keep track of the progress
// for creating a deployment.
// As of now, the condition can have one of the following state:
//   - Locked: The condition is locked as the reconciler is creating resources on Github
//   - Created: All the required resources are created on Github but there's still information missing
//   - Healthy: All the information needed by Github to represent the Deployment as active has been submitted
//   - Error: An error happened, the Reason attached to the condition should give more information.
const kGithubCondition conditions.ConditionType = "github"

// GithubReconciler reconciles a Workspace object that includes github-related deployment as annotations
type GithubReconciler struct {
	API *github.Client
	record.EventRecorder
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile the Workspace with an environment deployment on Github.
// This will add events & conditions to the workspace so that it's possible to keep
// track of the progress of creating a github deployment. Although it's a reconciliation loop like the
// others, it is completely isolated and run its own manager as anything happening here has no impact
// on the lifecycle of a workspace.
//
// For the most part, this reconciliation loop will read information off the workspace, it will read its state,
// and information required to create a Github Environment & Deployment.
//
// However, this reconciliation loop *will* add a Condition to the workspace to keep track of progress and also
// give an overview of what's going on with Github's deployment.
func (r *GithubReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var workspace sequencer.Workspace

	if err := r.Get(ctx, req.NamespacedName, &workspace); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("E#5001: Couldn't retrieve the workspace (%s) -- %w", req.NamespacedName, err)
	}

	if !r.githubAnnotationConfigured(workspace.Annotations) {
		logger.Info("Workspace doesn't have github annotations set, ignoring", "Workspace", workspace.Name)
		return ctrl.Result{}, nil
	}

	// Reaching this stage means one of five things:
	//	1. The Workspace was created with Github annotations but has not been processed yet;
	//	2. The Workspace was already processed once but more information is needed;
	//	3. The Workspace has an error, or this condition had an error, and there's nothing to do;
	//	4. The Github Deployment is completed and there's nothing to do;
	//	5. The Workspace was deleted, and the finalizer needs to be removed.

	if workspace.Status.Phase == workspaces.PhaseTerminating {
		// The Workspace is marked for deletion and Github's annotation were present
		// so the Finalizer needs to be removed. To remove the finalizer, the reconciler will try to
		// mark the Deployment linked to this Workspace as inactive. There's many reason why this could fail and
		// while some could possibly be fixed, it becomes quite complicated to account for external services to be 100% configured.
		// During creation, an error message can be surfaced back to the user but during deletion, it becomes a blocking operation
		// as the Finalizer wouldn't be removed if this was a hard dependency.
		// For this reason, the reconciler will attempt the deletion only once, and it will **log any error** that prevents the cleanup on Github.
		// However, events/conditions won't be updated as we're in a deletion process and the Workspace will be removed from the API Server and all
		// the information will be lost.
		err := r.deleteDeployment(ctx, &workspace)

		if err != nil {
			r.Event(&workspace, core.EventTypeWarning, string(kGithubCondition), err.Error())
			controllerutil.RemoveFinalizer(&workspace, kGithubFinalizer)
			return ctrl.Result{}, r.Patch(ctx, &workspace, client.Merge)
		}

		return ctrl.Result{}, nil
	}

	condition := conditions.FindCondition(workspace.Status.Conditions, kGithubCondition)
	if condition == nil {
		// Condition being nil means there's enough information to start a deployment on Github
		// but the workspace has not been processed until now. Creating the condition is part of creating
		// a new deployment on Github for this Workspace.
		// At this point, it's safe to assume that the annotations were not analysed, the finalizer is not set
		// and there was no attempt beforehand at creating those.

		conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   kGithubCondition,
			Status: conditions.ConditionLocked,
			Reason: "Github Deployment required",
		})

		if err := r.Status().Patch(ctx, &workspace, client.Merge); err != nil {
			return ctrl.Result{}, err
		}

		env, err := r.getEnvironment(ctx, &workspace)

		if err != nil {
			r.Event(&workspace, core.EventTypeWarning, string(kGithubCondition), err.Error())
			conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
				Type:   kGithubCondition,
				Status: conditions.ConditionError,
				Reason: err.Error(),
			})
			return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.Merge)
		}

		if _, exists := workspace.Annotations[kGithubDeploymentKey]; !exists {
			var deployment *github.Deployment
			deployment, err = r.createDeployment(ctx, &workspace, env)

			if err != nil {
				r.Event(&workspace, core.EventTypeWarning, string(kGithubCondition), err.Error())
				conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
					Type:   kGithubCondition,
					Status: conditions.ConditionError,
					Reason: err.Error(),
				})
				return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.Merge)
			}

			workspace.Annotations[kGithubDeploymentKey] = strconv.FormatInt(deployment.GetID(), 10)
			if err = r.Patch(ctx, &workspace, client.Merge); err != nil {
				return ctrl.Result{}, err
			}
		}

		controllerutil.AddFinalizer(&workspace, kGithubFinalizer)

		conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   kGithubCondition,
			Status: conditions.ConditionCreated,
			Reason: "Deployment created, waiting on Workspace to complete",
		})

		if err = r.Patch(ctx, &workspace, client.Merge); err != nil {
			return ctrl.Result{}, err
		}

		if err = r.Status().Patch(ctx, &workspace, client.Merge); err != nil {
			return ctrl.Result{}, err
		}

		// If the reconcilation loop has reached this point, it means that the condition is set to Created, the
		// deployment was succesfully created on Github and the Deployment ID is properly set as the annotation if
		// it didn't exists.
		_, err = r.createDeploymentStatus(ctx, &workspace, &github.DeploymentStatusRequest{
			State: github.String("in_progress"),
		})

		return ctrl.Result{}, err
	}

	if condition.Status == conditions.ConditionLocked {
		// This is a fatal error, this means that a reconciliation loop had an error
		// while locking the resource but could not update the Workspace to unlock
		// the condition. It's impossible to know what was done and what wasn't so the
		// safest route is to mark this condition as errored and exit this loop.
		// The Deployment has failed and when the workspace terminates, it will
		// try to clean up if the finalizer exists and enough information exists
		// on the Workspace to clean everything.
		conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   kGithubCondition,
			Status: conditions.ConditionError,
			Reason: "The Condition was never unlocked",
		})

		_, err := r.createDeploymentStatus(ctx, &workspace, &github.DeploymentStatusRequest{
			State:        github.String("error"),
			Description:  github.String("An internal issue with Sequencer prevented the deployment to complete"),
			AutoInactive: github.Bool(false),
		})

		if err != nil {
			// Error creating the deployment status should be retried as nothing was persisted yet
			// so it's safe to just re-enqueue
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}

		return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.Merge)
	}

	if condition.Status == conditions.ConditionCreated {
		// The deployment was created and the workspace was progressing the last
		// time this ran, each time the reconciler hits this branch, it will attempt
		// to come to a conclusion. Either the deployment was successful and the deployment status
		// needs to be updated with the final data, or it failed and the deployment status
		// needs to be marked as failed.
		//
		// If a deployment ID is present in the annotation, let's created a deployment status
		// to let Github know this deployment has failed. An error for this condition can mean
		// that the deployment failed (bad auth token, transient error, etc.), but it can also
		// mean that the Workspace itself has failed. If the Workspace failed, let's
		// try to find the condition that led to the error and add this information to the deployment
		// on Github to let the user know the root cause.
		var err error

		switch workspace.Status.Phase {

		case workspaces.PhaseHealthy:
			// The workspace is deployed in Kubernetes and is healthy. All the information needed
			// to update the Github Deployment Status is available.
			_, err = r.createDeploymentStatus(ctx, &workspace, &github.DeploymentStatusRequest{
				State:          github.String("success"),
				EnvironmentURL: &workspace.Status.Host,
				AutoInactive:   github.Bool(false),
			})

			if err != nil {
				// Error creating the deployment status should be retried as nothing was persisted yet
				// so it's safe to just re-enqueue
				return ctrl.Result{RequeueAfter: time.Second * 10}, err
			}

			conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
				Type:   kGithubCondition,
				Status: conditions.ConditionHealthy,
				Reason: "Github Deployment Status updated",
			})

			return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.Merge)

		case workspaces.PhaseError:
			// The Workspace had and
			condition := conditions.FindStatusCondition(workspace.Status.Conditions, conditions.ConditionError)
			if condition == nil {
				condition = &conditions.Condition{
					Status: conditions.ConditionError,
					Reason: "Unknown error, this is a bug in Sequencer, please file an issue: https://github.com/pier-oliviert/sequencer/issues",
				}
			}

			_, err = r.createDeploymentStatus(ctx, &workspace, &github.DeploymentStatusRequest{
				State:        github.String("error"),
				AutoInactive: github.Bool(false),
				Description:  github.String(condition.Reason),
			})

			if err != nil {
				// Error creating the deployment status should be retried as nothing was persisted yet
				// so it's safe to just re-enqueue
				return ctrl.Result{RequeueAfter: time.Second * 10}, err
			}
		}

		return ctrl.Result{}, r.Status().Patch(ctx, &workspace, client.Merge)
	}

	// Reaching here means that condition was either Error or Healthy, both cases mean there's
	// nothing to do anymore.
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sequencer.Workspace{}).
		Complete(r)
}

func (r *GithubReconciler) createDeploymentStatus(ctx context.Context, workspace *sequencer.Workspace, request *github.DeploymentStatusRequest) (*github.DeploymentStatus, error) {
	id, err := strconv.ParseInt(workspace.Annotations[kGithubDeploymentKey], 10, 64)
	if err != nil {
		return nil, err
	}

	status, _, err := r.API.Repositories.CreateDeploymentStatus(ctx, workspace.Annotations[kGithubOwnerKey], workspace.Annotations[kGithubRepositoryKey], id, request)

	return status, err
}

func (r *GithubReconciler) createDeployment(ctx context.Context, workspace *sequencer.Workspace, env *github.Environment) (*github.Deployment, error) {
	request := &github.DeploymentRequest{
		Environment: github.String(*env.Name),
		Ref:         github.String(workspace.Annotations[kGithubRefKey]),
	}
	deployment, _, err := r.API.Repositories.CreateDeployment(ctx, workspace.Annotations[kGithubOwnerKey], workspace.Annotations[kGithubRepositoryKey], request)

	return deployment, err
}

func (r *GithubReconciler) getDeployment(ctx context.Context, workspace *sequencer.Workspace) (*github.Deployment, error) {
	id, err := strconv.ParseInt(workspace.Annotations[kGithubDeploymentKey], 10, 64)
	if err != nil {
		return nil, err
	}

	deployment, _, err := r.API.Repositories.GetDeployment(ctx, workspace.Annotations[kGithubOwnerKey], workspace.Annotations[kGithubRepositoryKey], id)

	return deployment, err
}

func (r *GithubReconciler) getEnvironment(ctx context.Context, workspace *sequencer.Workspace) (*github.Environment, error) {
	env, _, err := r.API.Repositories.GetEnvironment(ctx, workspace.Annotations[kGithubOwnerKey], workspace.Annotations[kGithubRepositoryKey], workspace.Annotations[kGithubEnvironmentKey])
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (r *GithubReconciler) deleteDeployment(ctx context.Context, workspace *sequencer.Workspace) error {
	id, err := strconv.ParseInt(workspace.Annotations[kGithubDeploymentKey], 10, 64)
	if err != nil {
		return err
	}

	_, err = r.API.Repositories.DeleteDeployment(ctx, workspace.Annotations[kGithubOwnerKey], workspace.Annotations[kGithubRepositoryKey], id)
	return err
}

// Returns true if all the annotation *required* are set.
func (r *GithubReconciler) githubAnnotationConfigured(annotations map[string]string) bool {
	if value, found := annotations[kGithubEnvironmentKey]; !found {
		return false
	} else if value == "" {
		return false
	}

	if value, found := annotations[kGithubRefKey]; !found {
		return false
	} else if value == "" {
		return false
	}

	if value, found := annotations[kGithubOwnerKey]; !found {
		return false
	} else if value == "" {
		return false
	}

	if value, found := annotations[kGithubRepositoryKey]; !found {
		return false
	} else if value == "" {
		return false
	}

	return true
}
