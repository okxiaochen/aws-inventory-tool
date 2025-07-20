package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// LambdaCollector collects Lambda functions
type LambdaCollector struct {
	clientManager *awspkg.ClientManager
}

// NewLambdaCollector creates a new Lambda collector
func NewLambdaCollector(clientManager *awspkg.ClientManager) *LambdaCollector {
	return &LambdaCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *LambdaCollector) Name() string {
	return "lambda"
}

// Regions returns the regions this collector supports
func (c *LambdaCollector) Regions() []string {
	// Lambda is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves Lambda functions for the given region
func (c *LambdaCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := lambda.NewFromConfig(cfg)

	var resources []models.Resource
	var marker *string

	for {
		input := &lambda.ListFunctionsInput{
			Marker: marker,
		}

		result, err := client.ListFunctions(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list functions in %s: %w", region, err)
		}

		for _, function := range result.Functions {
			resource := c.convertFunction(function, region)
			resources = append(resources, resource)
		}

		marker = result.NextMarker
		if marker == nil {
			break
		}
	}

	return resources, nil
}

// convertFunction converts a Lambda function to a Resource
func (c *LambdaCollector) convertFunction(function types.FunctionConfiguration, region string) models.Resource {
	resource := models.Resource{
		Service: "lambda",
		Region:  region,
		ID:      aws.ToString(function.FunctionName),
		Name:    aws.ToString(function.FunctionName),
		Type:    string(function.Runtime),
		State:   string(function.State),
		Class:   fmt.Sprintf("%dMB", function.MemorySize),
	}

	// Set creation time
	if function.LastModified != nil {
		// Parse the time string
		if lastModified, err := time.Parse(time.RFC3339, *function.LastModified); err == nil {
			resource.CreatedAt = &lastModified
		}
	}

	// Add extra information
	extra := make(map[string]interface{})
	if function.FunctionArn != nil {
		extra["functionArn"] = aws.ToString(function.FunctionArn)
	}
	if function.Description != nil {
		extra["description"] = aws.ToString(function.Description)
	}
	if function.Handler != nil {
		extra["handler"] = aws.ToString(function.Handler)
	}
	if function.CodeSize > 0 {
		extra["codeSize"] = function.CodeSize
	}
	if function.Timeout != nil {
		extra["timeout"] = aws.ToInt32(function.Timeout)
	}
	if function.MemorySize != nil {
		extra["memorySize"] = aws.ToInt32(function.MemorySize)
	}
	if function.Version != nil {
		extra["version"] = aws.ToString(function.Version)
	}
	if function.Environment != nil && function.Environment.Variables != nil {
		extra["environmentVariables"] = len(function.Environment.Variables)
	}
	// Note: ReservedConcurrentExecutions is not available in this version
	if function.LastUpdateStatus != "" {
		extra["lastUpdateStatus"] = string(function.LastUpdateStatus)
	}
	if function.PackageType != "" {
		extra["packageType"] = string(function.PackageType)
	}
	if function.Architectures != nil {
		architectures := make([]string, len(function.Architectures))
		for i, arch := range function.Architectures {
			architectures[i] = string(arch)
		}
		extra["architectures"] = architectures
	}

	resource.Extra = extra

	return resource
} 