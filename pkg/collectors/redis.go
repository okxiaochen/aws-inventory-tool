package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// RedisCollector collects ElastiCache Redis clusters
type RedisCollector struct {
	clientManager *awspkg.ClientManager
}

// NewRedisCollector creates a new Redis collector
func NewRedisCollector(clientManager *awspkg.ClientManager) *RedisCollector {
	return &RedisCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *RedisCollector) Name() string {
	return "redis"
}

// Regions returns the regions this collector supports
func (c *RedisCollector) Regions() []string {
	// ElastiCache is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves ElastiCache Redis clusters for the given region
func (c *RedisCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := elasticache.NewFromConfig(cfg)

	var resources []models.Resource
	var marker *string

	for {
		input := &elasticache.DescribeCacheClustersInput{
			Marker: marker,
		}

		result, err := client.DescribeCacheClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe cache clusters in %s: %w", region, err)
		}

		for _, cluster := range result.CacheClusters {
			// Only collect Redis clusters
			if cluster.Engine != nil && aws.ToString(cluster.Engine) == "redis" {
				resource := c.convertCacheCluster(cluster, region)
				resources = append(resources, resource)
			}
		}

		marker = result.Marker
		if marker == nil {
			break
		}
	}

	return resources, nil
}

// convertCacheCluster converts an ElastiCache cluster to a Resource
func (c *RedisCollector) convertCacheCluster(cluster types.CacheCluster, region string) models.Resource {
	resource := models.Resource{
		Service: "redis",
		Region:  region,
		ID:      aws.ToString(cluster.CacheClusterId),
		Name:    aws.ToString(cluster.CacheClusterId),
		Type:    aws.ToString(cluster.Engine),
		State:   aws.ToString(cluster.CacheClusterStatus),
		Class:   aws.ToString(cluster.CacheNodeType),
	}

	// Set creation time
	if cluster.CacheClusterCreateTime != nil {
		createdAt := aws.ToTime(cluster.CacheClusterCreateTime)
		resource.CreatedAt = &createdAt
	}

	// Add extra information
	extra := make(map[string]interface{})
	if cluster.EngineVersion != nil {
		extra["engineVersion"] = aws.ToString(cluster.EngineVersion)
	}
	if cluster.ConfigurationEndpoint != nil {
		extra["endpoint"] = aws.ToString(cluster.ConfigurationEndpoint.Address)
		extra["port"] = cluster.ConfigurationEndpoint.Port
	}
	if cluster.PreferredAvailabilityZone != nil {
		extra["availabilityZone"] = aws.ToString(cluster.PreferredAvailabilityZone)
	}
	if cluster.NumCacheNodes != nil {
		extra["numCacheNodes"] = aws.ToInt32(cluster.NumCacheNodes)
	}
	if cluster.CacheParameterGroup != nil {
		extra["parameterGroup"] = aws.ToString(cluster.CacheParameterGroup.CacheParameterGroupName)
	}
	if cluster.CacheSubnetGroupName != nil {
		extra["subnetGroup"] = aws.ToString(cluster.CacheSubnetGroupName)
	}
	if cluster.SecurityGroups != nil {
		extra["securityGroups"] = cluster.SecurityGroups
	}
	if cluster.AtRestEncryptionEnabled != nil {
		extra["atRestEncryption"] = aws.ToBool(cluster.AtRestEncryptionEnabled)
	}
	if cluster.TransitEncryptionEnabled != nil {
		extra["transitEncryption"] = aws.ToBool(cluster.TransitEncryptionEnabled)
	}
	if cluster.AuthTokenEnabled != nil {
		extra["authTokenEnabled"] = aws.ToBool(cluster.AuthTokenEnabled)
	}
	if cluster.ReplicationGroupId != nil {
		extra["replicationGroupId"] = aws.ToString(cluster.ReplicationGroupId)
	}
	if cluster.SnapshotRetentionLimit != nil {
		extra["snapshotRetentionLimit"] = aws.ToInt32(cluster.SnapshotRetentionLimit)
	}
	if cluster.SnapshotWindow != nil {
		extra["snapshotWindow"] = aws.ToString(cluster.SnapshotWindow)
	}
	if cluster.PreferredMaintenanceWindow != nil {
		extra["maintenanceWindow"] = aws.ToString(cluster.PreferredMaintenanceWindow)
	}
	if cluster.AutoMinorVersionUpgrade != nil {
		extra["autoMinorVersionUpgrade"] = aws.ToBool(cluster.AutoMinorVersionUpgrade)
	}

	resource.Extra = extra

	return resource
} 