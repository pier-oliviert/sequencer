package providers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
)

const kAWSZoneID = "AWS_ZONE_ID"
const kAWSHostedZoneID = "AWS_HOSTED_ZONE_ID"
const kAWSLoadBalancerHost = "AWS_LOAD_BALANCER_HOST"

type r53 struct {
	hostedZoneID     string
	zoneID           string
	loadBalancerHost string
	*route53.Client
}

func NewAWSProvider() (*r53, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	zoneID, err := retrieveValueFromEnvOrFile(kAWSZoneID)
	if err != nil {
		return nil, fmt.Errorf("E#4102: Zone ID not found -- %w", err)
	}

	hostedZoneID, err := retrieveValueFromEnvOrFile(kAWSHostedZoneID)
	if err != nil {
		return nil, fmt.Errorf("E#4102: Hosted Zone ID not found -- %w", err)
	}

	loadBalancerHost, err := retrieveValueFromEnvOrFile(kAWSLoadBalancerHost)
	if err != nil {
		return nil, fmt.Errorf("E#4102: Load balancer host not found -- %w", err)
	}

	return &r53{
		zoneID:           zoneID,
		hostedZoneID:     hostedZoneID,
		loadBalancerHost: loadBalancerHost,
		Client:           route53.NewFromConfig(cfg),
	}, nil
}

func (c *r53) Create(ctx context.Context, record *sequencer.DNSRecord) error {
	inputs := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &c.zoneID,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action:            types.ChangeActionCreate,
				ResourceRecordSet: c.resourceRecordSet(record),
			}},
		},
	}

	_, err := c.ChangeResourceRecordSets(ctx, &inputs)
	return err
}

func (c *r53) Delete(ctx context.Context, record *sequencer.DNSRecord) error {
	inputs := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &c.zoneID,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action:            types.ChangeActionDelete,
				ResourceRecordSet: c.resourceRecordSet(record),
			}},
		},
	}

	_, err := c.ChangeResourceRecordSets(ctx, &inputs)
	return err
}

// Convert a DNSRecord to a resourceRecordSet
func (c *r53) resourceRecordSet(record *sequencer.DNSRecord) *types.ResourceRecordSet {
	set := types.ResourceRecordSet{
		Name: &record.Spec.Name,
		Type: types.RRType(record.Spec.RecordType),
	}

	if set.Type == types.RRTypeA || set.Type == types.RRTypeCname {
		set.AliasTarget = &types.AliasTarget{
			DNSName:      &record.Spec.Target,
			HostedZoneId: &c.hostedZoneID,
		}
	} else {
		set.ResourceRecords = append(set.ResourceRecords, types.ResourceRecord{
			Value: &record.Spec.Target,
		})
		set.TTL = new(int64)
		*set.TTL = 60

	}

	return &set
}
