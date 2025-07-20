package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// ECSCollector collects ECS clusters and services
type ECSCollector struct {
	clientManager *awspkg.ClientManager
}

// NewECSCollector creates a new ECS collector
func NewECSCollector(clientManager *awspkg.ClientManager) *ECSCollector {
	return &ECSCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *ECSCollector) Name() string {
	return "ecs"
}

// Regions returns the regions this collector supports
func (c *ECSCollector) Regions() []string {
	// ECS is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves ECS clusters and services for the given region
func (c *ECSCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := ecs.NewFromConfig(cfg)

	var resources []models.Resource
	var nextToken *string

	// List clusters
	for {
		input := &ecs.ListClustersInput{
			NextToken: nextToken,
		}

		result, err := client.ListClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters in %s: %w", region, err)
		}

		// Get detailed information for each cluster
		for _, clusterArn := range result.ClusterArns {
			clusterArnStr := aws.ToString(&clusterArn)
			clusterInfo, err := c.getClusterInfo(ctx, client, clusterArnStr)
			if err != nil {
				// Log error but continue with other clusters
				fmt.Printf("Warning: failed to get info for cluster %s: %v\n", clusterArnStr, err)
				continue
			}
			resource := c.convertCluster(clusterInfo, region)
			resources = append(resources, resource)

			// Also collect services in this cluster
			services, err := c.getClusterServices(ctx, client, clusterArnStr, region)
			if err != nil {
				fmt.Printf("Warning: failed to get services for cluster %s: %v\n", clusterArnStr, err)
				continue
			}
			resources = append(resources, services...)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return resources, nil
}

// getClusterInfo retrieves detailed information about an ECS cluster
func (c *ECSCollector) getClusterInfo(ctx context.Context, client *ecs.Client, clusterArn string) (*types.Cluster, error) {
	input := &ecs.DescribeClustersInput{
		Clusters: []string{clusterArn},
	}

	result, err := client.DescribeClusters(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Clusters) == 0 {
		return nil, fmt.Errorf("cluster not found: %s", clusterArn)
	}

	return &result.Clusters[0], nil
}

// getClusterServices retrieves services for a cluster
func (c *ECSCollector) getClusterServices(ctx context.Context, client *ecs.Client, clusterArn string, region string) ([]models.Resource, error) {
	var resources []models.Resource
	var nextToken *string

	for {
		input := &ecs.ListServicesInput{
			Cluster:   aws.String(clusterArn),
			NextToken: nextToken,
		}

		result, err := client.ListServices(ctx, input)
		if err != nil {
			return nil, err
		}

		// Get detailed information for each service
		for _, serviceArn := range result.ServiceArns {
			serviceArnStr := aws.ToString(&serviceArn)
			serviceInfo, err := c.getServiceInfo(ctx, client, serviceArnStr, clusterArn)
			if err != nil {
				fmt.Printf("Warning: failed to get info for service %s: %v\n", serviceArnStr, err)
				continue
			}
			resource := c.convertService(serviceInfo, region)
			resources = append(resources, resource)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return resources, nil
}

// getServiceInfo retrieves detailed information about an ECS service
func (c *ECSCollector) getServiceInfo(ctx context.Context, client *ecs.Client, serviceArn string, clusterArn string) (*types.Service, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterArn),
		Services: []string{serviceArn},
	}

	result, err := client.DescribeServices(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Services) == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceArn)
	}

	return &result.Services[0], nil
}

// convertCluster converts an ECS cluster to a Resource
func (c *ECSCollector) convertCluster(cluster *types.Cluster, region string) models.Resource {
	resource := models.Resource{
		Service: "ecs",
		Region:  region,
		ID:      aws.ToString(cluster.ClusterName),
		Name:    aws.ToString(cluster.ClusterName),
		Type:    "cluster",
		State:   aws.ToString(cluster.Status),
		Class:   "cluster",
	}

	// Note: ECS clusters don't have a CreatedAt field in this version
	// resource.CreatedAt = nil

	// Add extra information
	extra := make(map[string]interface{})
	if cluster.ClusterArn != nil {
		extra["clusterArn"] = aws.ToString(cluster.ClusterArn)
	}
	if cluster.RunningTasksCount > 0 {
		extra["runningTasks"] = cluster.RunningTasksCount
	}
	if cluster.PendingTasksCount > 0 {
		extra["pendingTasks"] = cluster.PendingTasksCount
	}
	if cluster.RegisteredContainerInstancesCount > 0 {
		extra["registeredInstances"] = cluster.RegisteredContainerInstancesCount
	}
	if cluster.ActiveServicesCount > 0 {
		extra["activeServices"] = cluster.ActiveServicesCount
	}
	if cluster.CapacityProviders != nil {
		extra["capacityProviders"] = len(cluster.CapacityProviders)
	}
	if cluster.DefaultCapacityProviderStrategy != nil {
		extra["defaultCapacityStrategy"] = len(cluster.DefaultCapacityProviderStrategy)
	}

	resource.Extra = extra

	return resource
}

// convertService converts an ECS service to a Resource
func (c *ECSCollector) convertService(service *types.Service, region string) models.Resource {
	resource := models.Resource{
		Service: "ecs",
		Region:  region,
		ID:      aws.ToString(service.ServiceName),
		Name:    aws.ToString(service.ServiceName),
		Type:    "service",
		State:   aws.ToString(service.Status),
		Class:   string(service.LaunchType),
	}

	// Note: ECS services don't have a CreatedAt field in this version
	// resource.CreatedAt = nil

	// Add extra information
	extra := make(map[string]interface{})
	if service.ServiceArn != nil {
		extra["serviceArn"] = aws.ToString(service.ServiceArn)
	}
	if service.ClusterArn != nil {
		extra["clusterArn"] = aws.ToString(service.ClusterArn)
	}
	if service.DesiredCount > 0 {
		extra["desiredCount"] = service.DesiredCount
	}
	if service.RunningCount > 0 {
		extra["runningCount"] = service.RunningCount
	}
	if service.PendingCount > 0 {
		extra["pendingCount"] = service.PendingCount
	}
	if service.LaunchType != "" {
		extra["launchType"] = string(service.LaunchType)
	}
	if service.TaskDefinition != nil {
		extra["taskDefinition"] = aws.ToString(service.TaskDefinition)
	}
	if service.NetworkConfiguration != nil {
		extra["networkMode"] = "awsvpc"
	}
	if service.LoadBalancers != nil {
		extra["loadBalancers"] = len(service.LoadBalancers)
	}
	if service.ServiceRegistries != nil {
		extra["serviceRegistries"] = len(service.ServiceRegistries)
	}

	resource.Extra = extra

	return resource
} 