package nameservers

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/workspaces"
	"se.quencer.io/internal/integrations"
	ctrl "sigs.k8s.io/controller-runtime"
)

const kAWSRoute53Finalizer = "dns.se.quencer.io/AWSRoute53"

type r53 struct {
	elbclient   *elb.Client
	r53client   *route53.Client
	credentials *credentialProvider
	integrations.ProviderController
}

func newAWSProvider(ctx context.Context, controller integrations.ProviderController) (integrations.Provider, error) {
	spec := controller.Workspace().Spec.Networking.AWS
	reconciler := &r53{
		ProviderController: controller,
	}

	// For some reason the alias LoadOptionsFunc doesn't work here, maybe some type
	// mismatch in the library
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(spec.Region),
	}

	if spec.SecretRef != nil {
		var secret core.Secret
		key := types.NamespacedName{
			Name: spec.SecretRef.Name,
		}

		if spec.SecretRef.Namespace != nil {
			key.Namespace = *spec.SecretRef.Namespace
		} else {
			key.Namespace = controller.Namespace()
		}

		if err := controller.Get(ctx, key, &secret); err != nil {
			return nil, err
		}

		if len(secret.Data["accessKeyID"]) == 0 {
			return nil, errors.New("E#4001: Using secret for the credentials, but `accessKeyID` is not present in the secret")
		}

		if len(secret.Data["secretAccessKey"]) == 0 {
			return nil, errors.New("E#4001: Using secret for the credentials, but `secretAccessKey` is not present in the secret")
		}

		reconciler.credentials = &credentialProvider{
			secret: secret,
		}

		opts = append(opts, config.WithCredentialsProvider(reconciler.credentials))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)

	if err != nil {
		return nil, fmt.Errorf("E#4001: AWS client could not be initialized -- %w", err)
	}

	reconciler.r53client = route53.NewFromConfig(cfg)
	reconciler.elbclient = elb.NewFromConfig(cfg)

	return reconciler, nil
}

func (r r53) Reconcile(ctx context.Context) (*ctrl.Result, error) {
	if r.Condition().Status != conditions.ConditionInitialized {
		return nil, nil
	}

	spec := r.Workspace().Spec.Networking
	host := fmt.Sprintf("%s.%s", r.Workspace().Name, spec.AWS.Route53.HostedZoneName)
	wildcard := fmt.Sprintf("*.%s.%s", r.Workspace().Name, spec.AWS.Route53.HostedZoneName)

	output, err := r.elbclient.DescribeLoadBalancers(ctx, &elb.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{spec.AWS.Route53.LoadBalancerArn},
	})

	if err != nil {
		return &ctrl.Result{}, fmt.Errorf("E#4002: Could not get info for AWS Load balancers -- %w", err)
	}

	if len(output.LoadBalancers) == 0 {
		return &ctrl.Result{}, errors.New("E#4002: Successfully received info from AWS, but no Load balancers present")
	}

	loadBalancer := output.LoadBalancers[0]

	batch := &rtypes.ChangeBatch{
		Comment: new(string),
		Changes: []rtypes.Change{{
			Action: rtypes.ChangeActionCreate,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name: &wildcard,
				AliasTarget: &rtypes.AliasTarget{
					HostedZoneId: loadBalancer.CanonicalHostedZoneId,
					DNSName:      loadBalancer.DNSName,
				},
				Type: rtypes.RRTypeA,
			},
		},
			{
				Action: rtypes.ChangeActionCreate,
				ResourceRecordSet: &rtypes.ResourceRecordSet{
					Name: &host,
					AliasTarget: &rtypes.AliasTarget{
						HostedZoneId: loadBalancer.CanonicalHostedZoneId,
						DNSName:      loadBalancer.DNSName,
					},
					Type: rtypes.RRTypeA,
				},
			},
		},
	}

	*batch.Comment = fmt.Sprintf("DNS records managed by Sequencer for %s", r.Workspace().Name)

	opts := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  batch,
		HostedZoneId: &spec.AWS.Route53.HostedZoneID,
	}

	if err := r.SetFinalizer(ctx, kAWSRoute53Finalizer); err != nil {
		return &ctrl.Result{}, err
	}

	err = r.Guard(ctx, "Modifying DNS records", func() (conditions.ConditionStatus, string, error) {

		response, err := r.r53client.ChangeResourceRecordSets(ctx, opts)
		if err != nil {
			return conditions.ConditionError, "Error submitting DNS records", err
		}

		status := &workspaces.DNS{}
		status.Provider = "route53"
		status.ProviderMeta = map[string]string{
			"ChangeID": *response.ChangeInfo.Id,
		}
		status.Hostname = host
		r.Workspace().Status.DNS = status

		return conditions.ConditionCreated, "Creating DNS records", nil
	})

	return &ctrl.Result{}, err
}

func (r r53) Terminate(ctx context.Context) (_ *ctrl.Result, err error) {
	spec := r.Workspace().Spec.Networking
	host := fmt.Sprintf("%s.%s", r.Workspace().Name, spec.AWS.Route53.HostedZoneName)
	wildcard := fmt.Sprintf("*.%s.%s", r.Workspace().Name, spec.AWS.Route53.HostedZoneName)

	output, err := r.elbclient.DescribeLoadBalancers(ctx, &elb.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{spec.AWS.Route53.LoadBalancerArn},
	})

	if err != nil {
		return &ctrl.Result{}, fmt.Errorf("E#4002: Could not get info for AWS Load balancers -- %w", err)
	}

	if len(output.LoadBalancers) == 0 {
		return &ctrl.Result{}, errors.New("E#4002: Successfully received info from AWS, but no Load balancers present")
	}

	// AWS Requires all the same values when deleting, this is why the comments
	// are also present in this changeBatch
	batch := &rtypes.ChangeBatch{
		Comment: new(string),
		Changes: []rtypes.Change{{
			Action: rtypes.ChangeActionDelete,
			ResourceRecordSet: &rtypes.ResourceRecordSet{
				Name: &wildcard,
				AliasTarget: &rtypes.AliasTarget{
					HostedZoneId: output.LoadBalancers[0].CanonicalHostedZoneId,
					DNSName:      output.LoadBalancers[0].DNSName,
				},
				Type: rtypes.RRTypeA,
			},
		},
			{
				Action: rtypes.ChangeActionDelete,
				ResourceRecordSet: &rtypes.ResourceRecordSet{
					Name: &host,
					AliasTarget: &rtypes.AliasTarget{
						HostedZoneId: output.LoadBalancers[0].CanonicalHostedZoneId,
						DNSName:      output.LoadBalancers[0].DNSName,
					},
					Type: rtypes.RRTypeA,
				},
			},
		},
	}
	*batch.Comment = fmt.Sprintf("DNS records managed by Sequencer for %s", r.Workspace().Name)

	opts := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  batch,
		HostedZoneId: &spec.AWS.Route53.HostedZoneID,
	}

	err = r.Guard(ctx, "Modifying DNS records", func() (conditions.ConditionStatus, string, error) {
		_, err := r.r53client.ChangeResourceRecordSets(ctx, opts)
		if err != nil {
			return conditions.ConditionError, "Error deleting DNS records", err
		}

		return conditions.ConditionTerminated, "Creating DNS records", nil
	})

	if err == nil {
		return &ctrl.Result{}, r.RemoveFinalizer(ctx, kAWSRoute53Finalizer)
	}

	return &ctrl.Result{}, err
}

type credentialProvider struct {
	secret core.Secret
}

func (cp credentialProvider) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     string(cp.secret.Data["accessKeyID"]),
		SecretAccessKey: string(cp.secret.Data["secretAccessKey"]),
		Source:          "User Secret Reference",
	}, nil
}
