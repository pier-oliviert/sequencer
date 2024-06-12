package providers

import "se.quencer.io/api/v1alpha1/utils"

// +kubebuilder:object:generate=true
type AWSSpec struct {
	Route53 *Route53Spec `json:"route53,omitempty"`

	// AWS Region (eg. us-east-2) where you want to operator in.
	Region string `json:"region"`

	// Reference to a Secret that contains the AWS Credentials. It needs two keys
	//	- accessKeyID
	//	- secretAccessKey
	//
	// This field is optional has the default configuration is to connect
	// a service account as described on AWS: https://docs.aws.amazon.com/eks/latest/userguide/pod-configuration.html
	// This field is made available for debugging purposes or for users that cannot use a service account.
	//
	// By default, this spec will attempt to load the AWS credential from the service account
	// but if this value is set, it will not check for AWS credential on the pod and
	// automatically use this value.
	SecretRef *utils.SecretRef `json:"secretRef,omitempty"`
}

type Route53Spec struct {
	HostedZoneID   string `json:"hostedZoneId"`
	HostedZoneName string `json:"hostedZoneName"`

	// There's no way currently to safely infer the load balancer that is configured
	// by the ingress controller. For that reason, the load balancer ARN needs to be defined
	// here, the LoadBalancer needs to be a Network Load Balancer(NLB) existing in the same region as
	// the one specified in the AWSSpec.
	//
	// The load balancer should alread be configured to route requests
	// to the Ingress controller that runs in the cluster
	LoadBalancerArn string `json:"loadBalancerArn"`
}
