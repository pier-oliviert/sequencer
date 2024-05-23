package k8s

import (
	"context"
	"fmt"
	"os"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	sequencer "se.quencer.io/api/v1alpha1"
	builds "se.quencer.io/api/v1alpha1/builds"
	"se.quencer.io/api/v1alpha1/conditions"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Client struct {
	broadcaster record.EventBroadcaster
	recorder    record.EventRecorder
	*rest.RESTClient
}

// Create a new Client that can communicate with the k8s cluster.
// This client will use the pod's service account to connect to the cluster
// and so requires read-write-list permissions on the Build CRD.
func NewClient(ctx context.Context, groupVersion *schema.GroupVersion) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentConfig.GroupVersion = groupVersion
	config.APIPath = "/apis"

	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	// This client is only used for events, because it seems like both clients
	// do not support all the features required so this needs to be setup so the broadcaster can push events.
	eventsClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Setting the broadcaster so events can be broadcasted back to the build.
	// Any events that are worth broadcasting will be part of the Build's Event to
	// be consumed by the user.
	broadcaster := record.NewBroadcaster()
	broadcaster.StartStructuredLogging(4)
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: eventsClient.CoreV1().Events(""),
	})

	return &Client{
		broadcaster,
		broadcaster.NewRecorder(scheme.Scheme, core.EventSource{Component: "Build"}),
		client,
	}, nil
}

func (c *Client) Close() {
	c.broadcaster.Shutdown()
}

// Return a Build custom resource from the k8s cluster. The build holds all the information
// to be able to build an image.
// If, for a reason or another, it can't retrieve the build set by the reference, this method
// will panic and terminate the process.
// Since it requires a k8s resource(*sequencer.Build) to record an event, it's not possible to gracefully communicate with
// the operator. It is the operator's responsibility, in this case, to either retry, or mark the build as errored.
func (c *Client) GetBuild(ctx context.Context, references []string) (*sequencer.Build, error) {
	logger := log.FromContext(ctx)

	if len(references) != 2 {
		return nil, fmt.Errorf("BUILD_REFERENCE is expected to have 2 components, had %d: %s", len(references), os.Getenv("BUILD_REFERENCE"))
	}

	logger.Info("Retrieving build CRD:", "references", references)

	var build sequencer.Build
	req := c.Get().Resource("builds").Namespace(references[0]).Name(references[1])
	result := req.Do(ctx)

	if err := result.Error(); err != nil {
		return nil, fmt.Errorf("error trying to get the build CRD: %v", err)
	}

	if err := result.Into(&build); err != nil {
		return nil, fmt.Errorf("error trying format the build: %v", err)
	}

	return &build, nil
}

// Same as GetBuild but panic if an error is returned.
func (c *Client) MustGetBuild(ctx context.Context, references []string) *sequencer.Build {
	b, err := c.GetBuild(ctx, references)
	if err != nil {
		panic(err)
	}

	return b
}

// Task represents a function that execute operation on a build.
// If an error occur while executing the task, it is the function's responsibility
// to return it so that the function calling this task have the opportunity of
// setting states on the build before continuing.
type Task func(Tracker) error

// StageCondition is used to stage a specific condition for a given +Build+ and +Task+. This is a convenience method
// to create a StagedCondition that allows the user to set some condition so it's easier to manage and isolate
// workload to the condition that is being staged.
//
// This method returns a StagedCondition, but doesn't persist anything up to this point.
// To do actual work, the user of this StagedCondition need to invoke +Do+.
func (c *Client) StageCondition(build *sequencer.Build, conditionType conditions.ConditionType) StagedCondition {
	condition := conditions.FindStatusCondition(build.Status.Conditions, conditionType)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   conditionType,
			Status: conditions.ConditionUnknown,
			Reason: builds.ConditionReasonInitialized,
		}
	}

	return StagedCondition{
		build:     build,
		client:    c,
		condition: *condition,
	}
}

type StagedCondition struct {
	build     *sequencer.Build
	client    *Client
	condition conditions.Condition
}

// Do runs the task to completion and attempts to update the condition staged
// for this Build. If the task returns an error, this method will call
// +tracker.Error(err)+ which means the condition will be set to +ConditionError+ and
// the builder will panic.
//
// If successful, the tracker will attempt to update the staged condition to
// +ConditionCompleted+.
//
// If any error returns from attempting an update of the staged condition, the
// method will panic.
func (sc StagedCondition) Do(ctx context.Context, task Task) {
	tracker := Tracker{
		build:     sc.build,
		client:    sc.client,
		condition: sc.condition,
		ctx:       ctx,
	}

	if err := task(tracker); err != nil {
		tracker.Error(err)
	}

	if err := tracker.Update(conditions.ConditionCompleted, builds.ConditionReasonCompleted); err != nil {
		tracker.Error(err)
	}
}

// Tracker is an opaque type that includes all the information from a StagedCondition. Some
// convenience method exists to have an easy way to interact with a condition. Any of the methods
// available that interact with a condition _commits_ the information to Kubernetes backend. It's
// designed that way as operator needs to be idempotent and updates to any condition is considered
// as a idempotency checkpoint.
type Tracker struct {
	ctx       context.Context
	build     *sequencer.Build
	client    *Client
	condition conditions.Condition
}

// Returns the context that was given to the StagedCondition.
func (t Tracker) Context() context.Context {
	return t.ctx
}

// Record a reason and a message to the broadcast recorder. This is not guaranteed
// to make it to the kubernetes event backend. It's useful to provide a high overview log
// that is attached to the Build custom resource.
func (t Tracker) Record(reason, message string) {
	t.client.recorder.Event(t.build, core.EventTypeNormal, reason, message)
}

// Error are fatal within the builder. When an error happens, the builder uses a best-attempt approach to try to
// record a warning message then fail the staged condition. At the end, it panics.
func (t Tracker) Error(err error) {
	t.client.recorder.Event(t.build, core.EventTypeWarning, string(t.condition.Type), err.Error())

	if anotherErr := t.Update(conditions.ConditionError, err.Error()); anotherErr != nil {
		t.client.recorder.Event(t.build, core.EventTypeWarning, "Status Error", anotherErr.Error())
		t.client.closeAndPanic(anotherErr)
	}

	t.client.closeAndPanic(err)
}

// Updates the staged condition to the status provided with the reason given. It commits the condition to the
// BuildStatus so this will do a roundtrip to the kubernetes backend. This is like a checkpoint for the build process.
func (t Tracker) Update(status conditions.ConditionStatus, reason string) error {
	conditions.SetStatusCondition(&t.build.Status.Conditions, conditions.Condition{
		Type:   t.condition.Type,
		Status: status,
		Reason: reason,
	})

	return t.client.updateBuildStatus(t.Context(), t.build)
}

func (c *Client) updateBuildStatus(ctx context.Context, build *sequencer.Build) error {
	result := c.Put().Resource("builds").SubResource("status").Namespace(build.Namespace).Name(build.Name).Body(build).Do(ctx)
	if err := result.Error(); err != nil {
		c.closeAndPanic(err)
	}

	if err := result.Into(build); err != nil {
		return err
	}

	return nil
}

func (c *Client) closeAndPanic(err error) {
	c.Close()
	panic(err)
}
