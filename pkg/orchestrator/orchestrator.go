package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/collectors"
	"github.com/xiaochen/awsinv/pkg/models"
)

// Orchestrator manages the collection of AWS resources across services and regions
type Orchestrator struct {
	clientManager *awspkg.ClientManager
	collectors    map[string]models.Collector
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(clientManager *awspkg.ClientManager) *Orchestrator {
	o := &Orchestrator{
		clientManager: clientManager,
		collectors:    make(map[string]models.Collector),
	}

	// Register all collectors
	o.registerCollectors()

	return o
}

// registerCollectors registers all available collectors
func (o *Orchestrator) registerCollectors() {
	o.collectors["ec2"] = collectors.NewEC2Collector(o.clientManager)
	o.collectors["rds"] = collectors.NewRDSCollector(o.clientManager)
	o.collectors["lambda"] = collectors.NewLambdaCollector(o.clientManager)
	o.collectors["s3"] = collectors.NewS3Collector(o.clientManager)
	o.collectors["dynamodb"] = collectors.NewDynamoDBCollector(o.clientManager)
	o.collectors["sfn"] = collectors.NewSFNCollector(o.clientManager)
	o.collectors["cloudwatch"] = collectors.NewCloudWatchCollector(o.clientManager)
	o.collectors["ecs"] = collectors.NewECSCollector(o.clientManager)
	o.collectors["redis"] = collectors.NewRedisCollector(o.clientManager)
	o.collectors["efs"] = collectors.NewEFSCollector(o.clientManager)
}

// GetAvailableServices returns the list of available services
func (o *Orchestrator) GetAvailableServices() []string {
	var services []string
	for service := range o.collectors {
		services = append(services, service)
	}
	sort.Strings(services)
	return services
}

// CollectOptions holds options for the collection process
type CollectOptions struct {
	Services   []string
	Regions    []string
	Parallel   int
	FailFast   bool
	Timeout    time.Duration
	Verbose    bool
}

// Collect performs the inventory collection across all specified services and regions
func (o *Orchestrator) Collect(ctx context.Context, opts CollectOptions) (*models.ResourceCollection, error) {
	startTime := time.Now()

	// Validate and prepare services
	services, err := o.prepareServices(opts.Services)
	if err != nil {
		return nil, err
	}

	// Discover or validate regions
	regions, err := o.prepareRegions(ctx, opts.Regions)
	if err != nil {
		return nil, err
	}

	// Create work items
	workItems := o.createWorkItems(services, regions)

	// Execute collection
	results := o.executeCollection(ctx, workItems, opts)

	// Aggregate results
	collection := o.aggregateResults(results, startTime)

	return collection, nil
}

// prepareServices validates and prepares the list of services to collect
func (o *Orchestrator) prepareServices(services []string) ([]string, error) {
	if len(services) == 0 {
		// Return all available services
		return o.GetAvailableServices(), nil
	}

	var validServices []string
	var invalidServices []string

	for _, service := range services {
		if _, exists := o.collectors[service]; exists {
			validServices = append(validServices, service)
		} else {
			invalidServices = append(invalidServices, service)
		}
	}

	if len(invalidServices) > 0 {
		return nil, fmt.Errorf("invalid services: %s", strings.Join(invalidServices, ", "))
	}

	return validServices, nil
}

// prepareRegions discovers or validates regions
func (o *Orchestrator) prepareRegions(ctx context.Context, regions []string) ([]string, error) {
	if len(regions) == 0 {
		// Discover all regions
		return o.clientManager.DiscoverRegions(ctx)
	}

	// Validate provided regions
	return o.clientManager.ValidateRegions(ctx, regions)
}

// workItem represents a single collection task
type workItem struct {
	Service string
	Region  string
}

// createWorkItems creates work items for all service-region combinations
func (o *Orchestrator) createWorkItems(services, regions []string) []workItem {
	var items []workItem

	for _, service := range services {
		collector := o.collectors[service]
		collectorRegions := collector.Regions()

		// If collector specifies regions, use those; otherwise use all regions
		if len(collectorRegions) > 0 {
			for _, region := range collectorRegions {
				items = append(items, workItem{Service: service, Region: region})
			}
		} else {
			for _, region := range regions {
				items = append(items, workItem{Service: service, Region: region})
			}
		}
	}

	return items
}

// executeCollection executes the collection in parallel
func (o *Orchestrator) executeCollection(ctx context.Context, workItems []workItem, opts CollectOptions) []models.CollectorResult {
	var results []models.CollectorResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create semaphore for parallel execution
	semaphore := make(chan struct{}, opts.Parallel)

	for _, item := range workItems {
		wg.Add(1)
		go func(item workItem) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			// Execute collection
			result := o.collectSingle(ctx, item, opts.Verbose)

			// Add result
			mu.Lock()
			results = append(results, result)
			mu.Unlock()

			// Handle fail-fast
			if opts.FailFast && result.Error != nil {
				// Cancel context to stop other goroutines
				// Note: This is a simplified approach; in production you might want more sophisticated cancellation
			}
		}(item)
	}

	wg.Wait()
	return results
}

// collectSingle collects resources for a single service-region combination
func (o *Orchestrator) collectSingle(ctx context.Context, item workItem, verbose bool) models.CollectorResult {
	collector := o.collectors[item.Service]

	if verbose && stderr != nil {
		if w, ok := stderr.(interface{ Write([]byte) (int, error) }); ok {
			fmt.Fprintf(w, "Collecting %s resources in %s...\n", item.Service, item.Region)
		}
	}

	resources, err := collector.Collect(ctx, item.Region)

	return models.CollectorResult{
		Service:   item.Service,
		Region:    item.Region,
		Resources: resources,
		Error:     err,
	}
}

// aggregateResults aggregates all collection results into a ResourceCollection
func (o *Orchestrator) aggregateResults(results []models.CollectorResult, startTime time.Time) *models.ResourceCollection {
	var allResources []models.Resource
	var errors []string
	summary := models.Summary{
		ByService: make(map[string]int),
		ByRegion:  make(map[string]int),
		ByState:   make(map[string]int),
		Duration:  time.Since(startTime),
	}

	// Track unique regions and services
	regionSet := make(map[string]bool)
	serviceSet := make(map[string]bool)

	for _, result := range results {
		if result.Error != nil {
			errorMsg := fmt.Sprintf("%s/%s: %v", result.Service, result.Region, result.Error)
			errors = append(errors, errorMsg)
			summary.Errors++
		} else {
			allResources = append(allResources, result.Resources...)
			
			// Update summary
			summary.ByService[result.Service] += len(result.Resources)
			summary.ByRegion[result.Region] += len(result.Resources)
			
			regionSet[result.Region] = true
			serviceSet[result.Service] = true

			// Count by state
			for _, resource := range result.Resources {
				if resource.State != "" {
					summary.ByState[resource.State]++
				}
			}
		}
	}

	// Convert sets to slices
	for region := range regionSet {
		summary.Regions = append(summary.Regions, region)
	}
	for service := range serviceSet {
		summary.Services = append(summary.Services, service)
	}

	summary.TotalResources = len(allResources)

	return &models.ResourceCollection{
		Resources: allResources,
		Errors:    errors,
		Summary:   summary,
	}
}

// stderr is used for verbose output
var stderr interface{} = nil

// SetStderr sets the stderr for verbose output
func SetStderr(w interface{}) {
	stderr = w
} 