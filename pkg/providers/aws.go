package providers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
)

const kAWSZoneID = "AWS_ZONE_ID"

type r53 struct {
	zoneID string
	*route53.Client
}

func NewAWSProvider() (*r53, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithClientLogMode(aws.LogRequestWithBody|aws.LogResponseWithBody))
	if err != nil {
		return nil, err
	}
	zoneID, err := retrieveValueFromEnvOrFile(kAWSZoneID)
	if err != nil {
		return nil, fmt.Errorf("E#6101: Zone ID not found -- %w", err)
	}

	return &r53{
		zoneID: zoneID,
		Client: route53.NewFromConfig(cfg),
	}, nil
}

func (c *r53) Create(ctx context.Context, record *sequencer.DNSRecord) error {
	inputs := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &c.zoneID,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action: types.ChangeActionCreate,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name: stringPtr(fmt.Sprintf("%s.%s", record.Spec.Name, record.Spec.Zone)),
					Type: types.RRType(record.Spec.RecordType),
					AliasTarget: &types.AliasTarget{
						DNSName:      &record.Spec.Target,
						HostedZoneId: &c.zoneID,
					},
				},
			}},
		},
	}

	_, err := c.Client.ChangeResourceRecordSets(ctx, &inputs)
	return err
}

func (c *r53) Delete(ctx context.Context, record *sequencer.DNSRecord) error {
	inputs := route53.ChangeResourceRecordSetsInput{
		HostedZoneId: &c.zoneID,
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{{
				Action: types.ChangeActionDelete,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name: stringPtr(fmt.Sprintf("%s.%s", record.Spec.Name, record.Spec.Zone)),
					Type: types.RRType(record.Spec.RecordType),
					AliasTarget: &types.AliasTarget{
						DNSName:      &record.Spec.Target,
						HostedZoneId: &c.zoneID,
					},
				},
			}},
		},
	}

	_, err := c.Client.ChangeResourceRecordSets(ctx, &inputs)
	return err
}

func stringPtr(val string) *string {
	return &val
}
