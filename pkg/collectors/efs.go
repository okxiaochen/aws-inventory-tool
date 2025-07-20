package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// EFSCollector collects EFS file systems
type EFSCollector struct {
	clientManager *aws.ClientManager
}

// NewEFSCollector creates a new EFS collector
func NewEFSCollector(clientManager *aws.ClientManager) *EFSCollector {
	return &EFSCollector{
		clientManager: clientManager,
	}
}

// Name returns the collector name
func (c *EFSCollector) Name() string {
	return "efs"
}

// Regions returns the regions this collector supports
func (c *EFSCollector) Regions() []string {
	return nil // EFS is available in all regions
}

// Collect discovers EFS file systems in the specified region
func (c *EFSCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	client, err := c.clientManager.GetEFSClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get EFS client: %w", err)
	}

	var resources []models.Resource

	// List file systems
	paginator := efs.NewDescribeFileSystemsPaginator(client, &efs.DescribeFileSystemsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe file systems: %w", err)
		}

		for _, fs := range page.FileSystems {
			resource := models.Resource{
				Service:   "efs",
				Region:    region,
				ID:        *fs.FileSystemId,
				Name:      getEFSName(fs),
				Type:      string(fs.PerformanceMode),
				State:     string(fs.LifeCycleState),
				Class:     string(fs.ThroughputMode),
				CreatedAt: fs.CreationTime,
				Tags:      convertEFSTags(fs.Tags),
				Extra: map[string]interface{}{
					"sizeBytes":        fs.SizeInBytes,
					"encrypted":        fs.Encrypted,
					"kmsKeyId":         fs.KmsKeyId,
					"availabilityZone": fs.AvailabilityZoneId,
				},
			}

			// Note: Mount targets would need separate API call to get
			// For now, we'll skip this to keep the collector simple

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// getEFSName extracts the name from EFS tags or uses ID
func getEFSName(fs types.FileSystemDescription) string {
	// Look for Name tag first
	for _, tag := range fs.Tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}

	// Look for common naming patterns in other tags
	for _, tag := range fs.Tags {
		if *tag.Key == "name" || *tag.Key == "Name" || *tag.Key == "displayName" {
			return *tag.Value
		}
	}

	// Use file system ID as fallback
	return *fs.FileSystemId
}

// convertEFSTags converts EFS tags to the standard format
func convertEFSTags(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[*tag.Key] = *tag.Value
	}
	return result
} 