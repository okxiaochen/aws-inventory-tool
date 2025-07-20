package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// S3Collector collects S3 buckets
type S3Collector struct {
	clientManager *awspkg.ClientManager
}

// NewS3Collector creates a new S3 collector
func NewS3Collector(clientManager *awspkg.ClientManager) *S3Collector {
	return &S3Collector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *S3Collector) Name() string {
	return "s3"
}

// Regions returns the regions this collector supports
func (c *S3Collector) Regions() []string {
	// S3 buckets are global, but we'll use us-east-1 for the API calls
	return []string{"us-east-1"}
}

// Collect retrieves S3 buckets
func (c *S3Collector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	// S3 buckets are global, so we use us-east-1 for the API calls
	cfg := c.clientManager.GetConfig("us-east-1")
	client := s3.NewFromConfig(cfg)

	var resources []models.Resource

	// S3 ListBuckets doesn't support pagination in the same way
	input := &s3.ListBucketsInput{}

	result, err := client.ListBuckets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	for _, bucket := range result.Buckets {
		resource := c.convertBucket(bucket)
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertBucket converts an S3 bucket to a Resource
func (c *S3Collector) convertBucket(bucket types.Bucket) models.Resource {
	resource := models.Resource{
		Service: "s3",
		Region:  "global", // S3 buckets are global
		ID:      aws.ToString(bucket.Name),
		Name:    aws.ToString(bucket.Name),
		Type:    "bucket",
		State:   "active", // Assume active if we can list it
	}

	// Set creation time
	if bucket.CreationDate != nil {
		createdAt := aws.ToTime(bucket.CreationDate)
		resource.CreatedAt = &createdAt
	}

	// Add extra information
	extra := make(map[string]interface{})
	if bucket.Name != nil {
		extra["bucketName"] = aws.ToString(bucket.Name)
	}

	resource.Extra = extra

	return resource
} 