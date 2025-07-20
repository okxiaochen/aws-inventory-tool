package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// EC2Collector collects EC2 instances
type EC2Collector struct {
	clientManager *awspkg.ClientManager
}

// NewEC2Collector creates a new EC2 collector
func NewEC2Collector(clientManager *awspkg.ClientManager) *EC2Collector {
	return &EC2Collector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *EC2Collector) Name() string {
	return "ec2"
}

// Regions returns the regions this collector supports
func (c *EC2Collector) Regions() []string {
	// EC2 is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves EC2 instances for the given region
func (c *EC2Collector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := ec2.NewFromConfig(cfg)

	var resources []models.Resource
	var nextToken *string

	for {
		input := &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		}

		result, err := client.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances in %s: %w", region, err)
		}

		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				resource := c.convertInstance(instance, region)
				resources = append(resources, resource)
			}
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return resources, nil
}

// convertInstance converts an EC2 instance to a Resource
func (c *EC2Collector) convertInstance(instance types.Instance, region string) models.Resource {
	resource := models.Resource{
		Service: "ec2",
		Region:  region,
		ID:      aws.ToString(instance.InstanceId),
		Type:    string(instance.InstanceType),
		State:   string(instance.State.Name),
	}

	// Extract name from tags
	if instance.Tags != nil {
		tags := make(map[string]string)
		for _, tag := range instance.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
				if aws.ToString(tag.Key) == "Name" {
					resource.Name = aws.ToString(tag.Value)
				}
			}
		}
		resource.Tags = tags
	}

	// Set creation time
	if instance.LaunchTime != nil {
		createdAt := aws.ToTime(instance.LaunchTime)
		resource.CreatedAt = &createdAt
	}

	// Add extra information
	extra := make(map[string]interface{})
	if instance.Platform != "" {
		extra["platform"] = string(instance.Platform)
	}
	if instance.Architecture != "" {
		extra["architecture"] = string(instance.Architecture)
	}
	if instance.RootDeviceType != "" {
		extra["rootDeviceType"] = string(instance.RootDeviceType)
	}
	if instance.VirtualizationType != "" {
		extra["virtualizationType"] = string(instance.VirtualizationType)
	}
	if instance.Hypervisor != "" {
		extra["hypervisor"] = string(instance.Hypervisor)
	}
	if instance.PrivateIpAddress != nil {
		extra["privateIp"] = aws.ToString(instance.PrivateIpAddress)
	}
	if instance.PublicIpAddress != nil {
		extra["publicIp"] = aws.ToString(instance.PublicIpAddress)
	}
	if instance.KeyName != nil {
		extra["keyName"] = aws.ToString(instance.KeyName)
	}

	resource.Extra = extra

	return resource
} 