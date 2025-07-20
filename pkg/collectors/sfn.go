package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// SFNCollector collects Step Functions state machines
type SFNCollector struct {
	clientManager *awspkg.ClientManager
}

// NewSFNCollector creates a new Step Functions collector
func NewSFNCollector(clientManager *awspkg.ClientManager) *SFNCollector {
	return &SFNCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *SFNCollector) Name() string {
	return "sfn"
}

// Regions returns the regions this collector supports
func (c *SFNCollector) Regions() []string {
	// Step Functions is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves Step Functions state machines for the given region
func (c *SFNCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := sfn.NewFromConfig(cfg)

	var resources []models.Resource
	var nextToken *string

	for {
		input := &sfn.ListStateMachinesInput{
			MaxResults: 100,
			NextToken:  nextToken,
		}

		result, err := client.ListStateMachines(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list state machines in %s: %w", region, err)
		}

		// Get detailed information for each state machine
		for _, stateMachine := range result.StateMachines {
			stateMachineInfo, err := c.getStateMachineInfo(ctx, client, aws.ToString(stateMachine.StateMachineArn))
			if err != nil {
				// Log error but continue with other state machines
				fmt.Printf("Warning: failed to get info for state machine %s: %v\n", aws.ToString(stateMachine.Name), err)
				continue
			}
			resource := c.convertStateMachine(stateMachineInfo, region)
			resources = append(resources, resource)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return resources, nil
}

// getStateMachineInfo retrieves detailed information about a Step Functions state machine
func (c *SFNCollector) getStateMachineInfo(ctx context.Context, client *sfn.Client, stateMachineArn string) (*sfn.DescribeStateMachineOutput, error) {
	input := &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String(stateMachineArn),
	}

	result, err := client.DescribeStateMachine(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// convertStateMachine converts a Step Functions state machine to a Resource
func (c *SFNCollector) convertStateMachine(stateMachine *sfn.DescribeStateMachineOutput, region string) models.Resource {
	resource := models.Resource{
		Service: "sfn",
		Region:  region,
		ID:      aws.ToString(stateMachine.Name),
		Name:    aws.ToString(stateMachine.Name),
		Type:    "state-machine",
		State:   string(stateMachine.Status),
		Class:   string(stateMachine.Type),
	}

	// Set creation time
	if stateMachine.CreationDate != nil {
		createdAt := aws.ToTime(stateMachine.CreationDate)
		resource.CreatedAt = &createdAt
	}

	// Add extra information
	extra := make(map[string]interface{})
	if stateMachine.StateMachineArn != nil {
		extra["stateMachineArn"] = aws.ToString(stateMachine.StateMachineArn)
	}
	if stateMachine.RoleArn != nil {
		extra["roleArn"] = aws.ToString(stateMachine.RoleArn)
	}
	if stateMachine.Definition != nil {
		extra["definitionLength"] = len(aws.ToString(stateMachine.Definition))
	}
	if stateMachine.Type != "" {
		extra["type"] = string(stateMachine.Type)
	}
	if stateMachine.LoggingConfiguration != nil {
		extra["loggingEnabled"] = stateMachine.LoggingConfiguration.Level != types.LogLevelOff
	}
	if stateMachine.TracingConfiguration != nil {
		extra["tracingEnabled"] = stateMachine.TracingConfiguration.Enabled
	}

	resource.Extra = extra

	return resource
} 