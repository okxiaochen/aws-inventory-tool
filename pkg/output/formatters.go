package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/xiaochen/awsinv/pkg/models"
)

// CostEstimate represents a cost estimate for a resource
type CostEstimate struct {
	Amount       float64
	Explanation  string
	Formula      string
	FormulaExplanation string
	Breakdown    map[string]float64
	Assumptions  []string
	Examples     []string
	Accuracy     string // "High", "Medium", "Low" - indicates estimation accuracy
}

// Formatter defines the interface for output formatters
type Formatter interface {
	Format(collection *models.ResourceCollection, filters []Filter, sortField string, noColor bool) error
}

// Filter represents a filter condition
type Filter struct {
	Key   string
	Value string
}

// ParseFilters parses filter strings in the format "key=value"
func ParseFilters(filterStrings []string) ([]Filter, error) {
	var filters []Filter

	for _, filterStr := range filterStrings {
		parts := strings.SplitN(filterStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid filter format: %s (expected key=value)", filterStr)
		}

		filters = append(filters, Filter{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}

	return filters, nil
}

// applyFilters applies filters to resources
func applyFilters(resources []models.Resource, filters []Filter) []models.Resource {
	if len(filters) == 0 {
		return resources
	}

	var filtered []models.Resource

	for _, resource := range resources {
		if matchesFilters(resource, filters) {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// matchesFilters checks if a resource matches all filters
func matchesFilters(resource models.Resource, filters []Filter) bool {
	for _, filter := range filters {
		if !matchesFilter(resource, filter) {
			return false
		}
	}
	return true
}

// matchesFilter checks if a resource matches a single filter
func matchesFilter(resource models.Resource, filter Filter) bool {
	var value string
	var isSubstring bool

	// Check if it's a substring match
	if strings.HasSuffix(filter.Value, "*") {
		value = strings.TrimSuffix(filter.Value, "*")
		isSubstring = true
	} else {
		value = filter.Value
		isSubstring = false
	}

	// Get the field value
	var fieldValue string
	switch filter.Key {
	case "service":
		fieldValue = resource.Service
	case "region":
		fieldValue = resource.Region
	case "id":
		fieldValue = resource.ID
	case "name":
		fieldValue = resource.Name
	case "type":
		fieldValue = resource.Type
	case "state":
		fieldValue = resource.State
	case "class":
		fieldValue = resource.Class
	default:
		// Check tags
		if tagValue, exists := resource.Tags[filter.Key]; exists {
			fieldValue = tagValue
		} else {
			return false
		}
	}

	// Perform comparison
	if isSubstring {
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(value))
	} else {
		return strings.EqualFold(fieldValue, value)
	}
}

// sortResources sorts resources by the specified field
func sortResources(resources []models.Resource, sortField string) {
	sort.Slice(resources, func(i, j int) bool {
		var a, b string

		switch sortField {
		case "service":
			a, b = resources[i].Service, resources[j].Service
		case "region":
			a, b = resources[i].Region, resources[j].Region
		case "id":
			a, b = resources[i].ID, resources[j].ID
		case "name":
			a, b = resources[i].Name, resources[j].Name
		case "type":
			a, b = resources[i].Type, resources[j].Type
		case "state":
			a, b = resources[i].State, resources[j].State
		default:
			a, b = resources[i].Service, resources[j].Service
		}

		if a == b {
			// Secondary sort by ID
			return resources[i].ID < resources[j].ID
		}
		return a < b
	})
}

// TableFormatter formats output as a table
type TableFormatter struct {
	writer *os.File
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(writer *os.File) *TableFormatter {
	return &TableFormatter{writer: writer}
}

// Format formats the collection as a table
func (f *TableFormatter) Format(collection *models.ResourceCollection, filters []Filter, sortField string, noColor bool) error {
	// Apply filters
	resources := applyFilters(collection.Resources, filters)

	// Sort resources
	sortResources(resources, sortField)

	// Calculate cost estimates
	costEstimates := calculateCostEstimates(resources)
	
	// Calculate total monthly cost
	totalMonthlyCost := 0.0
	for _, estimate := range costEstimates {
		if estimate != nil {
			totalMonthlyCost += estimate.Amount
		}
	}

	// Print summary
	fmt.Fprintf(f.writer, "\nAWS Resource Inventory Summary\n")
	fmt.Fprintf(f.writer, "==============================\n")
	fmt.Fprintf(f.writer, "Total Resources: %d\n", len(resources))
	fmt.Fprintf(f.writer, "Estimated Monthly Cost: $%.2f\n", totalMonthlyCost)
	fmt.Fprintf(f.writer, "Duration: %v\n", collection.Summary.Duration)
	fmt.Fprintf(f.writer, "Errors: %d\n", len(collection.Errors))

	if len(collection.Summary.ByService) > 0 {
		fmt.Fprintf(f.writer, "\nBy Service:\n")
		// Calculate service costs and create sorted list
		serviceCosts := make([]struct {
			Service string
			Count   int
			Cost    float64
		}, 0, len(collection.Summary.ByService))
		
		for service, count := range collection.Summary.ByService {
			serviceCost := 0.0
			for _, resource := range resources {
				if resource.Service == service {
					if estimate, exists := costEstimates[resource.ID]; exists && estimate != nil {
						serviceCost += estimate.Amount
					}
				}
			}
			serviceCosts = append(serviceCosts, struct {
				Service string
				Count   int
				Cost    float64
			}{service, count, serviceCost})
		}
		
		// Sort by cost (highest first)
		sort.Slice(serviceCosts, func(i, j int) bool {
			return serviceCosts[i].Cost > serviceCosts[j].Cost
		})
		
		for _, item := range serviceCosts {
			fmt.Fprintf(f.writer, "  %s: %d ($%.2f/month)\n", item.Service, item.Count, item.Cost)
		}
	}

	if len(collection.Summary.ByRegion) > 0 {
		fmt.Fprintf(f.writer, "\nBy Region:\n")
		for region, count := range collection.Summary.ByRegion {
			regionCost := 0.0
			for _, resource := range resources {
				if resource.Region == region {
					if estimate, exists := costEstimates[resource.ID]; exists && estimate != nil {
						regionCost += estimate.Amount
					}
				}
			}
			fmt.Fprintf(f.writer, "  %s: %d ($%.2f/month)\n", region, count, regionCost)
		}
	}



	// Print errors if any
	if len(collection.Errors) > 0 {
		fmt.Fprintf(f.writer, "\nErrors:\n")
		for _, err := range collection.Errors {
			fmt.Fprintf(f.writer, "  %s\n", err)
		}
	}

	// Print resources table
	if len(resources) > 0 {
		fmt.Fprintf(f.writer, "\nResources Inventory (Total Cost: $%.2f/month):\n", totalMonthlyCost)
		fmt.Fprintf(f.writer, "%-12s %-15s %-20s %-15s %-10s %-10s %-10s %-12s\n", "SERVICE", "REGION", "ID", "NAME", "TYPE", "STATE", "CLASS", "MONTHLY COST")
		fmt.Fprintf(f.writer, "%-12s %-15s %-20s %-15s %-10s %-10s %-10s %-12s\n", "-------", "------", "--", "----", "----", "-----", "-----", "------------")

		for _, resource := range resources {
			costStr := "-"
			if estimate, exists := costEstimates[resource.ID]; exists && estimate != nil {
				costStr = fmt.Sprintf("$%.2f", estimate.Amount)
			}
			
			fmt.Fprintf(f.writer, "%-12s %-15s %-20s %-15s %-10s %-10s %-10s %-12s\n",
				truncate(resource.Service, 12),
				truncate(resource.Region, 15),
				truncate(resource.ID, 20),
				truncate(resource.Name, 15),
				truncate(resource.Type, 10),
				truncate(resource.State, 10),
				truncate(resource.Class, 10),
				costStr)
		}
	}

	return nil
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	writer *os.File
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(writer *os.File) *JSONFormatter {
	return &JSONFormatter{writer: writer}
}

// ResourceWithCost represents a resource with its cost estimate
type ResourceWithCost struct {
	models.Resource
	CostEstimate *CostEstimate `json:"costEstimate,omitempty"`
}

// Format formats the collection as JSON
func (f *JSONFormatter) Format(collection *models.ResourceCollection, filters []Filter, sortField string, noColor bool) error {
	// Apply filters
	resources := applyFilters(collection.Resources, filters)

	// Sort resources
	sortResources(resources, sortField)

	// Calculate cost estimates
	costEstimates := calculateCostEstimates(resources)
	
	// Calculate total monthly cost
	totalMonthlyCost := 0.0
	for _, estimate := range costEstimates {
		if estimate != nil {
			totalMonthlyCost += estimate.Amount
		}
	}

	// Create resources with cost estimates
	resourcesWithCost := make([]ResourceWithCost, len(resources))
	for i, resource := range resources {
		resourcesWithCost[i] = ResourceWithCost{
			Resource: resource,
		}
		if estimate, exists := costEstimates[resource.ID]; exists && estimate != nil {
			resourcesWithCost[i].CostEstimate = estimate
		}
	}

	// Create output structure
	output := struct {
		Resources         []ResourceWithCost `json:"resources"`
		Summary           models.Summary     `json:"summary"`
		TotalMonthlyCost  float64            `json:"totalMonthlyCost"`
		Errors            []string           `json:"errors,omitempty"`
	}{
		Resources:        resourcesWithCost,
		Summary:          collection.Summary,
		TotalMonthlyCost: totalMonthlyCost,
		Errors:           collection.Errors,
	}

	// Update summary with filtered count
	output.Summary.TotalResources = len(resources)

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// CSVFormatter formats output as CSV
type CSVFormatter struct {
	writer *os.File
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter(writer *os.File) *CSVFormatter {
	return &CSVFormatter{writer: writer}
}

// Format formats the collection as CSV
func (f *CSVFormatter) Format(collection *models.ResourceCollection, filters []Filter, sortField string, noColor bool) error {
	// Apply filters
	resources := applyFilters(collection.Resources, filters)

	// Sort resources
	sortResources(resources, sortField)

	// Calculate cost estimates
	costEstimates := calculateCostEstimates(resources)

	writer := csv.NewWriter(f.writer)
	defer writer.Flush()

	// Write header
	header := []string{"Service", "Region", "ID", "Name", "Type", "State", "Class", "MonthlyCost", "CreatedAt", "Tags"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, resource := range resources {
		// Convert tags to string
		tagsStr := ""
		if len(resource.Tags) > 0 {
			var tagPairs []string
			for k, v := range resource.Tags {
				tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
			}
			tagsStr = strings.Join(tagPairs, ",")
		}

		// Convert creation time to string
		createdAtStr := ""
		if resource.CreatedAt != nil {
			createdAtStr = resource.CreatedAt.Format(time.RFC3339)
		}

		// Get cost estimate
		costStr := ""
		if estimate, exists := costEstimates[resource.ID]; exists && estimate != nil {
			costStr = fmt.Sprintf("%.2f", estimate.Amount)
		}

		row := []string{
			resource.Service,
			resource.Region,
			resource.ID,
			resource.Name,
			resource.Type,
			resource.State,
			resource.Class,
			costStr,
			createdAtStr,
			tagsStr,
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// stderr is used for verbose output
var stderr *os.File

// SetStderr sets the stderr file for verbose output
func SetStderr(file *os.File) {
	stderr = file
}

// calculateCostEstimates calculates cost estimates for individual resources
func calculateCostEstimates(resources []models.Resource) map[string]*CostEstimate {
	costs := make(map[string]*CostEstimate)

	for _, resource := range resources {
		var estimate *CostEstimate
		switch resource.Service {
		case "ec2":
			estimate = estimateEC2Cost(resource)
		case "rds":
			estimate = estimateRDSCost(resource)
		case "lambda":
			estimate = estimateLambdaCost(resource)
		case "s3":
			estimate = estimateS3Cost(resource)
		case "dynamodb":
			estimate = estimateDynamoDBCost(resource)
		case "sfn":
			estimate = estimateSFNCost(resource)
		case "cloudwatch":
			estimate = estimateCloudWatchCost(resource)
		case "ecs":
			estimate = estimateECSCost(resource)
		case "redis":
			estimate = estimateRedisCost(resource)
		default:
			estimate = &CostEstimate{Amount: 0}
		}
		
		if estimate != nil {
			costs[resource.ID] = estimate
		}
	}

	return costs
}

// estimateEC2Cost estimates EC2 instance cost (rough monthly estimate)
func estimateEC2Cost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      0,
		Explanation: "EC2 costs are based on instance type and running state",
		Formula:     "Monthly Cost = Hourly Rate × 730 hours",
		FormulaExplanation: "AWS charges per hour, so we multiply the hourly rate by 730 hours (average hours per month) to get monthly cost.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "High",
		Assumptions: []string{
			"Based on us-east-1 on-demand pricing",
			"Only running instances are charged",
			"Excludes data transfer, storage, and other costs",
			"Assumes 24/7 usage (730 hours/month)",
		},
		Examples: []string{
			"t3.micro: $0.0116/hour × 730 hours = $8.47/month",
			"t3.small: $0.0232/hour × 730 hours = $16.94/month",
			"m5.large: $0.1184/hour × 730 hours = $86.40/month",
		},
	}

	if resource.State != "running" {
		return estimate
	}

	// Rough cost estimates per month (us-east-1 pricing)
	costMap := map[string]float64{
		"t3.micro":     8.47,
		"t3.small":     16.94,
		"t3a.medium":   27.07,
		"t3.medium":    33.88,
		"t3.large":     67.76,
		"m5.large":     86.40,
		"m5.xlarge":    172.80,
		"c5.large":     68.00,
		"c5.xlarge":    136.00,
		"r5.large":     126.00,
		"r5.xlarge":    252.00,
	}

	if cost, exists := costMap[resource.Type]; exists {
		estimate.Amount = cost
		estimate.Breakdown[resource.Type] = cost
		estimate.Explanation = fmt.Sprintf("EC2 %s instance: $%.2f/month", resource.Type, cost)
	} else {
		estimate.Amount = 50.0
		estimate.Breakdown["unknown"] = 50.0
		estimate.Explanation = fmt.Sprintf("EC2 %s instance: $50.00/month (estimated for unknown instance type)", resource.Type)
		estimate.Assumptions = append(estimate.Assumptions, "Unknown instance type - using conservative estimate")
	}

	return estimate
}

// estimateRDSCost estimates RDS instance cost (rough monthly estimate)
func estimateRDSCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      0,
		Explanation: "RDS costs are based on instance class and availability",
		Formula:     "Monthly Cost = Hourly Rate × 730 hours",
		FormulaExplanation: "RDS instances are charged per hour, similar to EC2. We multiply the hourly rate by 730 hours for monthly cost.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "High",
		Assumptions: []string{
			"Based on us-east-1 on-demand pricing",
			"Only available instances are charged",
			"Excludes storage, backup, and data transfer costs",
			"Assumes 24/7 usage (730 hours/month)",
			"Single-AZ deployment pricing",
		},
		Examples: []string{
			"db.t3.micro: $0.0205/hour × 730 hours = $15.00/month",
			"db.m5.large: $0.234/hour × 730 hours = $171.00/month",
			"db.r5.large: $0.312/hour × 730 hours = $228.00/month",
		},
	}

	if resource.State != "available" {
		return estimate
	}

	// Rough cost estimates per month (us-east-1 pricing)
	costMap := map[string]float64{
		"db.t3.micro":    15.00,
		"db.t3.small":    30.00,
		"db.t3.medium":   60.00,
		"db.t3.large":    120.00,
		"db.m5.large":    171.00,
		"db.m5.xlarge":   342.00,
		"db.r5.large":    228.00,
		"db.r5.xlarge":   456.00,
	}

	if cost, exists := costMap[resource.Class]; exists {
		estimate.Amount = cost
		estimate.Breakdown[resource.Class] = cost
		estimate.Explanation = fmt.Sprintf("RDS %s instance: $%.2f/month", resource.Class, cost)
	} else {
		estimate.Amount = 100.0
		estimate.Breakdown["unknown"] = 100.0
		estimate.Explanation = fmt.Sprintf("RDS %s instance: $100.00/month (estimated for unknown instance class)", resource.Class)
		estimate.Assumptions = append(estimate.Assumptions, "Unknown instance class - using conservative estimate")
	}

	return estimate
}

// estimateLambdaCost estimates Lambda function cost (rough monthly estimate)
func estimateLambdaCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      5.0, // Conservative estimate
		Explanation: "Lambda costs are based on function execution and memory usage",
		Formula:     "Monthly Cost = $5.00 (estimated moderate usage)",
		FormulaExplanation: "Lambda pricing is complex (requests + duration + memory). Using conservative estimate for moderate usage.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "Medium",
		Assumptions: []string{
			"Estimated moderate usage (1000 requests/month)",
			"128MB memory allocation",
			"100ms average execution time",
			"Conservative estimate for unknown usage patterns",
		},
		Examples: []string{
			"Low usage: $1-3/month",
			"Moderate usage: $5-10/month",
			"High usage: $20-50/month",
		},
	}

	estimate.Breakdown["estimated"] = estimate.Amount
	estimate.Explanation = fmt.Sprintf("Lambda function %s: $%.2f/month (estimated)", resource.Name, estimate.Amount)

	return estimate
}

// estimateS3Cost estimates S3 bucket cost (rough monthly estimate)
func estimateS3Cost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      1.0, // Minimal usage estimate
		Explanation: "S3 costs are based on storage, requests, and data transfer",
		Formula:     "Monthly Cost = $1.00 (estimated minimal usage)",
		FormulaExplanation: "S3 pricing includes storage, requests, and data transfer. Using conservative estimate for minimal usage.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "Low",
		Assumptions: []string{
			"Estimated minimal usage (1GB storage)",
			"Standard storage class",
			"Low request volume",
			"Conservative estimate for unknown usage patterns",
		},
		Examples: []string{
			"Minimal usage: $1-3/month",
			"Moderate usage: $5-15/month",
			"High usage: $20-100/month",
		},
	}

	estimate.Breakdown["estimated"] = estimate.Amount
	estimate.Explanation = fmt.Sprintf("S3 bucket %s: $%.2f/month (estimated)", resource.Name, estimate.Amount)

	return estimate
}

// estimateDynamoDBCost estimates DynamoDB table cost (rough monthly estimate)
func estimateDynamoDBCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      10.0, // Conservative estimate
		Explanation: "DynamoDB costs are based on read/write capacity and storage",
		Formula:     "Monthly Cost = $10.00 (estimated moderate usage)",
		FormulaExplanation: "DynamoDB pricing includes read/write capacity units and storage. Using conservative estimate for moderate usage.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "Low",
		Assumptions: []string{
			"Estimated moderate read/write capacity",
			"On-demand billing mode",
			"Conservative estimate for unknown usage patterns",
		},
		Examples: []string{
			"Low usage: $5-10/month",
			"Moderate usage: $10-25/month",
			"High usage: $50-200/month",
		},
	}

	estimate.Breakdown["estimated"] = estimate.Amount
	estimate.Explanation = fmt.Sprintf("DynamoDB table %s: $%.2f/month (estimated)", resource.Name, estimate.Amount)

	return estimate
}

// estimateSFNCost estimates Step Functions cost (rough monthly estimate)
func estimateSFNCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      5.0, // Conservative estimate
		Explanation: "Step Functions costs are based on state transitions and execution time",
		Formula:     "Monthly Cost = $5.00 (estimated moderate usage)",
		FormulaExplanation: "Step Functions pricing is based on state transitions and execution time. Using conservative estimate for moderate usage.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "Low",
		Assumptions: []string{
			"Estimated moderate workflow complexity",
			"Standard workflow execution",
			"Conservative estimate for unknown usage patterns",
		},
		Examples: []string{
			"Low usage: $2-5/month",
			"Moderate usage: $5-15/month",
			"High usage: $20-100/month",
		},
	}

	estimate.Breakdown["estimated"] = estimate.Amount
	estimate.Explanation = fmt.Sprintf("Step Function %s: $%.2f/month (estimated)", resource.Name, estimate.Amount)

	return estimate
}

// estimateCloudWatchCost estimates CloudWatch cost (rough monthly estimate)
func estimateCloudWatchCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      2.0, // Conservative estimate
		Explanation: "CloudWatch costs are based on metrics, logs, and alarms",
		Formula:     "Monthly Cost = $2.00 (estimated moderate usage)",
		FormulaExplanation: "CloudWatch pricing includes metrics, logs, and alarms. Using conservative estimate for moderate usage.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "Low",
		Assumptions: []string{
			"Estimated moderate metric resolution",
			"Standard resolution metrics",
			"Conservative estimate for unknown usage patterns",
		},
		Examples: []string{
			"Low usage: $1-3/month",
			"Moderate usage: $2-8/month",
			"High usage: $10-50/month",
		},
	}

	estimate.Breakdown["estimated"] = estimate.Amount
	estimate.Explanation = fmt.Sprintf("CloudWatch %s: $%.2f/month (estimated)", resource.Name, estimate.Amount)

	return estimate
}

// estimateECSCost estimates ECS cost (rough monthly estimate)
func estimateECSCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      0,
		Explanation: "ECS costs depend on underlying infrastructure (EC2/Fargate)",
		Formula:     "Monthly Cost = Infrastructure costs + ECS management",
		FormulaExplanation: "ECS itself is free, but you pay for the underlying infrastructure (EC2 instances or Fargate tasks) plus ECS management overhead.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "Medium",
		Assumptions: []string{
			"ECS service management overhead",
			"Infrastructure costs handled separately",
			"Conservative estimate for management overhead",
		},
		Examples: []string{
			"Cluster management: $5-10/month",
			"Service management: $5-15/month",
			"Infrastructure: $50-500/month (depends on EC2/Fargate)",
		},
	}

	// Different estimates based on resource type
	switch resource.Type {
	case "cluster":
		estimate.Amount = 5.0 // Cluster management overhead
		estimate.Explanation = fmt.Sprintf("ECS cluster %s: $%.2f/month (management overhead)", resource.Name, estimate.Amount)
	case "service":
		estimate.Amount = 15.0 // Service management overhead
		estimate.Explanation = fmt.Sprintf("ECS service %s: $%.2f/month (management overhead)", resource.Name, estimate.Amount)
	default:
		estimate.Amount = 10.0 // Default estimate
		estimate.Explanation = fmt.Sprintf("ECS %s: $%.2f/month (estimated)", resource.Name, estimate.Amount)
	}

	estimate.Breakdown["management"] = estimate.Amount

	return estimate
}

// estimateRedisCost estimates Redis (ElastiCache) cost (rough monthly estimate)
func estimateRedisCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      0,
		Explanation: "Redis costs are based on node type and availability",
		Formula:     "Monthly Cost = Hourly Rate × 730 hours",
		FormulaExplanation: "ElastiCache Redis instances are charged per hour, similar to EC2. We multiply the hourly rate by 730 hours for monthly cost.",
		Breakdown:   make(map[string]float64),
		Accuracy:    "High",
		Assumptions: []string{
			"Based on us-east-1 on-demand pricing",
			"Only available instances are charged",
			"Excludes data transfer and backup costs",
			"Assumes 24/7 usage (730 hours/month)",
			"Single-node deployment pricing",
		},
		Examples: []string{
			"cache.t3.micro: $0.017/hour × 730 hours = $12.41/month",
			"cache.t3.small: $0.034/hour × 730 hours = $24.82/month",
			"cache.m5.large: $0.136/hour × 730 hours = $99.28/month",
		},
	}

	if resource.State != "available" {
		return estimate
	}

	// Rough cost estimates per month (us-east-1 pricing)
	costMap := map[string]float64{
		"cache.t3.micro":    12.41,
		"cache.t3.small":    24.82,
		"cache.t3.medium":   49.64,
		"cache.t3.large":    99.28,
		"cache.m5.large":    99.28,
		"cache.m5.xlarge":   198.56,
		"cache.r5.large":    145.60,
		"cache.r5.xlarge":   291.20,
		"cache.c5.large":    81.60,
		"cache.c5.xlarge":   163.20,
	}

	if cost, exists := costMap[resource.Class]; exists {
		estimate.Amount = cost
		estimate.Breakdown[resource.Class] = cost
		estimate.Explanation = fmt.Sprintf("Redis %s instance: $%.2f/month", resource.Class, cost)
	} else {
		estimate.Amount = 50.0
		estimate.Breakdown["unknown"] = 50.0
		estimate.Explanation = fmt.Sprintf("Redis %s instance: $50.00/month (estimated for unknown node type)", resource.Class)
		estimate.Assumptions = append(estimate.Assumptions, "Unknown node type - using conservative estimate")
	}

	return estimate
} 