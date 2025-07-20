package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// CloudWatchCollector collects CloudWatch alarms
type CloudWatchCollector struct {
	clientManager *awspkg.ClientManager
}

// NewCloudWatchCollector creates a new CloudWatch collector
func NewCloudWatchCollector(clientManager *awspkg.ClientManager) *CloudWatchCollector {
	return &CloudWatchCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *CloudWatchCollector) Name() string {
	return "cloudwatch"
}

// Regions returns the regions this collector supports
func (c *CloudWatchCollector) Regions() []string {
	// CloudWatch is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves CloudWatch alarms for the given region
func (c *CloudWatchCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := cloudwatch.NewFromConfig(cfg)

	var resources []models.Resource
	var nextToken *string

	for {
		input := &cloudwatch.DescribeAlarmsInput{
			NextToken: nextToken,
		}

		result, err := client.DescribeAlarms(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe alarms in %s: %w", region, err)
		}

		for _, alarm := range result.MetricAlarms {
			resource := c.convertAlarm(alarm, region)
			resources = append(resources, resource)
		}

		// Also collect composite alarms
		for _, alarm := range result.CompositeAlarms {
			resource := c.convertCompositeAlarm(alarm, region)
			resources = append(resources, resource)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return resources, nil
}

// convertAlarm converts a CloudWatch metric alarm to a Resource
func (c *CloudWatchCollector) convertAlarm(alarm types.MetricAlarm, region string) models.Resource {
	resource := models.Resource{
		Service: "cloudwatch",
		Region:  region,
		ID:      aws.ToString(alarm.AlarmName),
		Name:    aws.ToString(alarm.AlarmName),
		Type:    "metric-alarm",
		State:   string(alarm.StateValue),
		Class:   "metric",
	}

	// Set creation time (CloudWatch doesn't provide creation time, so we'll use empty)
	// resource.CreatedAt = nil

	// Add extra information
	extra := make(map[string]interface{})
	if alarm.AlarmArn != nil {
		extra["alarmArn"] = aws.ToString(alarm.AlarmArn)
	}
	if alarm.AlarmDescription != nil {
		extra["description"] = aws.ToString(alarm.AlarmDescription)
	}
	if alarm.MetricName != nil {
		extra["metricName"] = aws.ToString(alarm.MetricName)
	}
	if alarm.Namespace != nil {
		extra["namespace"] = aws.ToString(alarm.Namespace)
	}
	if alarm.Threshold != nil {
		extra["threshold"] = aws.ToFloat64(alarm.Threshold)
	}
	if alarm.ComparisonOperator != "" {
		extra["comparisonOperator"] = string(alarm.ComparisonOperator)
	}
	if alarm.EvaluationPeriods != nil {
		extra["evaluationPeriods"] = aws.ToInt32(alarm.EvaluationPeriods)
	}
	if alarm.Period != nil {
		extra["period"] = aws.ToInt32(alarm.Period)
	}
	if alarm.Statistic != "" {
		extra["statistic"] = string(alarm.Statistic)
	}
	if alarm.TreatMissingData != nil {
		extra["treatMissingData"] = string(*alarm.TreatMissingData)
	}
	if alarm.ActionsEnabled != nil {
		extra["actionsEnabled"] = aws.ToBool(alarm.ActionsEnabled)
	}
	if alarm.AlarmActions != nil {
		extra["alarmActions"] = len(alarm.AlarmActions)
	}
	if alarm.OKActions != nil {
		extra["okActions"] = len(alarm.OKActions)
	}
	if alarm.InsufficientDataActions != nil {
		extra["insufficientDataActions"] = len(alarm.InsufficientDataActions)
	}

	resource.Extra = extra

	return resource
}

// convertCompositeAlarm converts a CloudWatch composite alarm to a Resource
func (c *CloudWatchCollector) convertCompositeAlarm(alarm types.CompositeAlarm, region string) models.Resource {
	resource := models.Resource{
		Service: "cloudwatch",
		Region:  region,
		ID:      aws.ToString(alarm.AlarmName),
		Name:    aws.ToString(alarm.AlarmName),
		Type:    "composite-alarm",
		State:   string(alarm.StateValue),
		Class:   "composite",
	}

	// Add extra information
	extra := make(map[string]interface{})
	if alarm.AlarmArn != nil {
		extra["alarmArn"] = aws.ToString(alarm.AlarmArn)
	}
	if alarm.AlarmDescription != nil {
		extra["description"] = aws.ToString(alarm.AlarmDescription)
	}
	if alarm.AlarmRule != nil {
		extra["alarmRule"] = aws.ToString(alarm.AlarmRule)
	}
	if alarm.ActionsEnabled != nil {
		extra["actionsEnabled"] = aws.ToBool(alarm.ActionsEnabled)
	}
	if alarm.AlarmActions != nil {
		extra["alarmActions"] = len(alarm.AlarmActions)
	}
	if alarm.OKActions != nil {
		extra["okActions"] = len(alarm.OKActions)
	}
	if alarm.InsufficientDataActions != nil {
		extra["insufficientDataActions"] = len(alarm.InsufficientDataActions)
	}

	resource.Extra = extra

	return resource
} 