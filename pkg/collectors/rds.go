package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// RDSCollector collects RDS database instances
type RDSCollector struct {
	clientManager *awspkg.ClientManager
}

// NewRDSCollector creates a new RDS collector
func NewRDSCollector(clientManager *awspkg.ClientManager) *RDSCollector {
	return &RDSCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *RDSCollector) Name() string {
	return "rds"
}

// Regions returns the regions this collector supports
func (c *RDSCollector) Regions() []string {
	// RDS is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves RDS database instances for the given region
func (c *RDSCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := rds.NewFromConfig(cfg)

	var resources []models.Resource
	var marker *string

	for {
		input := &rds.DescribeDBInstancesInput{
			Marker: marker,
		}

		result, err := client.DescribeDBInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB instances in %s: %w", region, err)
		}

		for _, instance := range result.DBInstances {
			resource := c.convertDBInstance(instance, region)
			resources = append(resources, resource)
		}

		marker = result.Marker
		if marker == nil {
			break
		}
	}

	return resources, nil
}

// convertDBInstance converts an RDS DB instance to a Resource
func (c *RDSCollector) convertDBInstance(instance types.DBInstance, region string) models.Resource {
	resource := models.Resource{
		Service: "rds",
		Region:  region,
		ID:      aws.ToString(instance.DBInstanceIdentifier),
		Name:    aws.ToString(instance.DBInstanceIdentifier),
		Type:    aws.ToString(instance.Engine),
		State:   aws.ToString(instance.DBInstanceStatus),
		Class:   aws.ToString(instance.DBInstanceClass),
	}

	// Set creation time
	if instance.InstanceCreateTime != nil {
		createdAt := aws.ToTime(instance.InstanceCreateTime)
		resource.CreatedAt = &createdAt
	}

	// Add extra information
	extra := make(map[string]interface{})
	if instance.EngineVersion != nil {
		extra["engineVersion"] = aws.ToString(instance.EngineVersion)
	}
	if instance.Endpoint != nil {
		extra["endpoint"] = aws.ToString(instance.Endpoint.Address)
		extra["port"] = instance.Endpoint.Port
	}
	if instance.AvailabilityZone != nil {
		extra["availabilityZone"] = aws.ToString(instance.AvailabilityZone)
	}
	if instance.MultiAZ != nil {
		extra["multiAZ"] = aws.ToBool(instance.MultiAZ)
	}
	if instance.StorageEncrypted != nil {
		extra["storageEncrypted"] = aws.ToBool(instance.StorageEncrypted)
	}
	if instance.AllocatedStorage != nil {
		extra["allocatedStorage"] = aws.ToInt32(instance.AllocatedStorage)
	}
	if instance.MaxAllocatedStorage != nil {
		extra["maxAllocatedStorage"] = aws.ToInt32(instance.MaxAllocatedStorage)
	}
	if instance.StorageType != nil {
		extra["storageType"] = aws.ToString(instance.StorageType)
	}
	if instance.LicenseModel != nil {
		extra["licenseModel"] = aws.ToString(instance.LicenseModel)
	}
	if instance.DeletionProtection != nil {
		extra["deletionProtection"] = aws.ToBool(instance.DeletionProtection)
	}
	if instance.BackupRetentionPeriod != nil {
		extra["backupRetentionPeriod"] = aws.ToInt32(instance.BackupRetentionPeriod)
	}
	if instance.PreferredBackupWindow != nil {
		extra["preferredBackupWindow"] = aws.ToString(instance.PreferredBackupWindow)
	}
	if instance.PreferredMaintenanceWindow != nil {
		extra["preferredMaintenanceWindow"] = aws.ToString(instance.PreferredMaintenanceWindow)
	}

	resource.Extra = extra

	return resource
} 