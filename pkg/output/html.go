package output

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/xiaochen/awsinv/pkg/models"
)

// HTMLFormatter formats output as HTML
type HTMLFormatter struct {
	writer *os.File
}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter(writer *os.File) *HTMLFormatter {
	return &HTMLFormatter{writer: writer}
}

// Format formats the collection as HTML
func (f *HTMLFormatter) Format(collection *models.ResourceCollection, filters []Filter, sortField string, noColor bool) error {
	// Apply filters
	resources := applyFilters(collection.Resources, filters)

	// Sort resources
	sortResources(resources, sortField)

	// Calculate cost estimates
	costEstimates := calculateCostEstimates(resources)

	// Create template with custom functions
	funcMap := template.FuncMap{
		"add": func(a, b float64) float64 {
			return a + b
		},
		"addInt": func(a, b int) int {
			return a + b
		},
		"makeSlice": func() []interface{} {
			return []interface{}{}
		},
		"append": func(slice []interface{}, item interface{}) []interface{} {
			return append(slice, item)
		},
		"unique": func(items []interface{}) []string {
			seen := make(map[string]bool)
			var result []string
			for _, item := range items {
				if str, ok := item.(string); ok && !seen[str] {
					seen[str] = true
					result = append(result, str)
				}
			}
			return result
		},
		"upper": strings.ToUpper,
		"eq": func(a, b string) bool {
			return a == b
		},
		"dict": func(keyvals ...interface{}) map[string]interface{} {
			if len(keyvals)%2 != 0 {
				return nil
			}
			m := make(map[string]interface{})
			for i := 0; i < len(keyvals); i += 2 {
				key, ok := keyvals[i].(string)
				if !ok {
					return nil
				}
				m[key] = keyvals[i+1]
			}
			return m
		},
	}

	// Create HTML template with custom functions
	tmpl := template.Must(template.New("inventory").Funcs(funcMap).Parse(htmlTemplate))

	// Create resource data with cost estimates
	type ResourceWithCost struct {
		models.Resource
		CostEstimate *CostEstimate
	}
	
	var resourcesWithCost []ResourceWithCost
	for _, resource := range resources {
		var costEstimate *CostEstimate
		switch resource.Service {
		case "ec2":
			costEstimate = estimateEC2Cost(resource)
		case "rds":
			costEstimate = estimateRDSCost(resource)
		case "lambda":
			costEstimate = estimateLambdaCost(resource)
		case "s3":
			costEstimate = estimateS3Cost(resource)
		case "dynamodb":
			costEstimate = estimateDynamoDBCost(resource)
		case "sfn":
			costEstimate = estimateSFNCost(resource)
		case "cloudwatch":
			costEstimate = estimateCloudWatchCost(resource)
		case "ecs":
			costEstimate = estimateECSCost(resource)
		}
		
		resourcesWithCost = append(resourcesWithCost, ResourceWithCost{
			Resource:     resource,
			CostEstimate: costEstimate,
		})
	}

	// Prepare data for template
	data := struct {
		Resources     []ResourceWithCost
		Summary       models.Summary
		Errors        []string
		CostEstimates map[string]*CostEstimate
		GeneratedAt   time.Time
	}{
		Resources:     resourcesWithCost,
		Summary:       collection.Summary,
		Errors:        collection.Errors,
		CostEstimates: costEstimates,
		GeneratedAt:   time.Now(),
	}

	// Execute template
	return tmpl.Execute(f.writer, data)
}

// CostEstimate represents a cost estimate with explanation
type CostEstimate struct {
	Amount       float64
	Explanation  string
	Formula      string
	FormulaExplanation string
	Breakdown    map[string]float64
	Assumptions  []string
	Examples     []string
}

// calculateCostEstimates calculates rough cost estimates for resources
func calculateCostEstimates(resources []models.Resource) map[string]*CostEstimate {
	costs := make(map[string]*CostEstimate)

	for _, resource := range resources {
		switch resource.Service {
		case "ec2":
			if costs["ec2"] == nil {
				costs["ec2"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateEC2Cost(resource)
			costs["ec2"].Amount += estimate.Amount
			costs["ec2"].Breakdown[resource.ID] = estimate.Amount
			costs["ec2"].Explanation = estimate.Explanation
			costs["ec2"].Assumptions = append(costs["ec2"].Assumptions, estimate.Assumptions...)
		case "rds":
			if costs["rds"] == nil {
				costs["rds"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateRDSCost(resource)
			costs["rds"].Amount += estimate.Amount
			costs["rds"].Breakdown[resource.ID] = estimate.Amount
			costs["rds"].Explanation = estimate.Explanation
			costs["rds"].Assumptions = append(costs["rds"].Assumptions, estimate.Assumptions...)
		case "lambda":
			if costs["lambda"] == nil {
				costs["lambda"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateLambdaCost(resource)
			costs["lambda"].Amount += estimate.Amount
			costs["lambda"].Breakdown[resource.ID] = estimate.Amount
			costs["lambda"].Explanation = estimate.Explanation
			costs["lambda"].Assumptions = append(costs["lambda"].Assumptions, estimate.Assumptions...)
		case "s3":
			if costs["s3"] == nil {
				costs["s3"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateS3Cost(resource)
			costs["s3"].Amount += estimate.Amount
			costs["s3"].Breakdown[resource.ID] = estimate.Amount
			costs["s3"].Explanation = estimate.Explanation
			costs["s3"].Assumptions = append(costs["s3"].Assumptions, estimate.Assumptions...)
		case "dynamodb":
			if costs["dynamodb"] == nil {
				costs["dynamodb"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateDynamoDBCost(resource)
			costs["dynamodb"].Amount += estimate.Amount
			costs["dynamodb"].Breakdown[resource.ID] = estimate.Amount
			costs["dynamodb"].Explanation = estimate.Explanation
			costs["dynamodb"].Assumptions = append(costs["dynamodb"].Assumptions, estimate.Assumptions...)
		case "sfn":
			if costs["sfn"] == nil {
				costs["sfn"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateSFNCost(resource)
			costs["sfn"].Amount += estimate.Amount
			costs["sfn"].Breakdown[resource.ID] = estimate.Amount
			costs["sfn"].Explanation = estimate.Explanation
			costs["sfn"].Assumptions = append(costs["sfn"].Assumptions, estimate.Assumptions...)
		case "cloudwatch":
			if costs["cloudwatch"] == nil {
				costs["cloudwatch"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateCloudWatchCost(resource)
			costs["cloudwatch"].Amount += estimate.Amount
			costs["cloudwatch"].Breakdown[resource.ID] = estimate.Amount
			costs["cloudwatch"].Explanation = estimate.Explanation
			costs["cloudwatch"].Assumptions = append(costs["cloudwatch"].Assumptions, estimate.Assumptions...)
		case "ecs":
			if costs["ecs"] == nil {
				costs["ecs"] = &CostEstimate{Breakdown: make(map[string]float64), Assumptions: []string{}}
			}
			estimate := estimateECSCost(resource)
			costs["ecs"].Amount += estimate.Amount
			costs["ecs"].Breakdown[resource.ID] = estimate.Amount
			costs["ecs"].Explanation = estimate.Explanation
			costs["ecs"].Assumptions = append(costs["ecs"].Assumptions, estimate.Assumptions...)
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
		Amount:      5.0,
		Explanation: "Lambda costs are based on requests, duration, and memory",
		Formula:     "Monthly Cost = (Requests × $0.20/million) + (Duration × Memory × $0.0000166667/GB-second)",
		FormulaExplanation: "Lambda charges per request ($0.20 per million requests) plus compute time based on memory allocation and execution duration.",
		Breakdown:   make(map[string]float64),
		Assumptions: []string{
			"Estimated $5/month per function",
			"Assumes moderate usage (1000 requests/month)",
			"Assumes 128MB memory allocation",
			"Assumes 100ms average execution time",
			"Excludes data transfer and other costs",
		},
		Examples: []string{
			"1000 requests: 1000 × $0.20/1,000,000 = $0.20",
			"Compute time: 1000 × 0.1s × 0.128GB × $0.0000166667 = $0.21",
			"Total: $0.20 + $0.21 = $0.41 (estimated $5 for overhead)",
		},
	}

	estimate.Breakdown["requests"] = 2.0
	estimate.Breakdown["duration"] = 3.0
	estimate.Explanation = fmt.Sprintf("Lambda function %s: $5.00/month", resource.Name)

	return estimate
}

// estimateS3Cost estimates S3 bucket cost (rough monthly estimate)
func estimateS3Cost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      1.0,
		Explanation: "S3 costs are based on storage, requests, and data transfer",
		Formula:     "Monthly Cost = (Storage × $0.023/GB) + (Requests × $0.0004/1000) + Data Transfer",
		FormulaExplanation: "S3 charges for storage per GB, requests per 1000 operations, and data transfer out of AWS.",
		Breakdown:   make(map[string]float64),
		Assumptions: []string{
			"Estimated $1/month per bucket",
			"Assumes minimal storage (1GB)",
			"Assumes standard storage class",
			"Assumes low request volume",
			"Excludes data transfer and other costs",
		},
		Examples: []string{
			"Storage: 1GB × $0.023/GB = $0.023",
			"Requests: 1000 × $0.0004/1000 = $0.40",
			"Total: $0.023 + $0.40 = $0.42 (estimated $1 for overhead)",
		},
	}

	estimate.Breakdown["storage"] = 0.5
	estimate.Breakdown["requests"] = 0.5
	estimate.Explanation = fmt.Sprintf("S3 bucket %s: $1.00/month", resource.Name)

	return estimate
}

// estimateDynamoDBCost estimates DynamoDB table cost (rough monthly estimate)
func estimateDynamoDBCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      10.0,
		Explanation: "DynamoDB costs are based on read/write capacity and storage",
		Formula:     "Monthly Cost = (Read Units × $0.25/unit) + (Write Units × $1.25/unit) + (Storage × $0.25/GB)",
		FormulaExplanation: "DynamoDB charges for read capacity units, write capacity units, and storage per GB.",
		Breakdown:   make(map[string]float64),
		Assumptions: []string{
			"Estimated $10/month per table",
			"Assumes on-demand billing mode",
			"Assumes moderate read/write capacity",
			"Assumes minimal storage (1GB)",
			"Excludes data transfer and other costs",
		},
		Examples: []string{
			"Read capacity: 10 units × $0.25 = $2.50",
			"Write capacity: 5 units × $1.25 = $6.25",
			"Storage: 1GB × $0.25/GB = $0.25",
			"Total: $2.50 + $6.25 + $0.25 = $9.00 (estimated $10)",
		},
	}

	estimate.Breakdown["read_capacity"] = 4.0
	estimate.Breakdown["write_capacity"] = 4.0
	estimate.Breakdown["storage"] = 2.0
	estimate.Explanation = fmt.Sprintf("DynamoDB table %s: $10.00/month", resource.Name)

	return estimate
}

// estimateSFNCost estimates Step Functions cost (rough monthly estimate)
func estimateSFNCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      5.0,
		Explanation: "Step Functions costs are based on state transitions and execution time",
		Formula:     "Monthly Cost = (State Transitions × $0.025/1000) + (Duration × $0.00001/second)",
		FormulaExplanation: "Step Functions charges per 1000 state transitions and per second of execution time.",
		Breakdown:   make(map[string]float64),
		Assumptions: []string{
			"Estimated $5/month per state machine",
			"Assumes moderate execution frequency",
			"Assumes standard workflow complexity",
			"Excludes Lambda and other service costs",
		},
		Examples: []string{
			"State transitions: 1000 × $0.025/1000 = $0.025",
			"Execution time: 1000s × $0.00001/s = $0.01",
			"Total: $0.025 + $0.01 = $0.035 (estimated $5 for overhead)",
		},
	}

	estimate.Breakdown["transitions"] = 3.0
	estimate.Breakdown["execution"] = 2.0
	estimate.Explanation = fmt.Sprintf("Step Functions %s: $5.00/month", resource.Name)

	return estimate
}

// estimateCloudWatchCost estimates CloudWatch cost (rough monthly estimate)
func estimateCloudWatchCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      2.0,
		Explanation: "CloudWatch costs are based on metrics, logs, and alarms",
		Formula:     "Monthly Cost = (Metrics × $0.30/metric) + (Alarms × $0.50/alarm) + Log Storage",
		FormulaExplanation: "CloudWatch charges for custom metrics, alarms, and log storage. Basic metrics are free.",
		Breakdown:   make(map[string]float64),
		Assumptions: []string{
			"Estimated $2/month per alarm",
			"Assumes standard resolution metrics",
			"Assumes moderate metric volume",
			"Excludes log storage and other costs",
		},
		Examples: []string{
			"Custom metrics: 5 × $0.30 = $1.50",
			"Alarm: 1 × $0.50 = $0.50",
			"Total: $1.50 + $0.50 = $2.00",
		},
	}

	estimate.Breakdown["metrics"] = 1.0
	estimate.Breakdown["alarms"] = 1.0
	estimate.Explanation = fmt.Sprintf("CloudWatch alarm %s: $2.00/month", resource.Name)

	return estimate
}

// estimateECSCost estimates ECS cost (rough monthly estimate)
func estimateECSCost(resource models.Resource) *CostEstimate {
	estimate := &CostEstimate{
		Amount:      0,
		Explanation: "ECS costs depend on underlying infrastructure and task definitions",
		Formula:     "Monthly Cost = (Fargate vCPU × $0.04048/hour) + (Fargate Memory × $0.004445/GB-hour) + EC2 Costs",
		FormulaExplanation: "ECS charges depend on whether you use Fargate (serverless) or EC2 instances. Fargate charges per vCPU and memory.",
		Breakdown:   make(map[string]float64),
		Assumptions: []string{
			"Costs are primarily from underlying EC2 instances or Fargate",
			"Cluster management is free",
			"Service costs depend on task definitions",
			"Excludes data transfer and other costs",
		},
		Examples: []string{
			"Fargate: 0.5 vCPU × $0.04048 × 730 hours = $14.78",
			"Memory: 1GB × $0.004445 × 730 hours = $3.24",
			"Total: $14.78 + $3.24 = $18.02 (estimated $15)",
		},
	}

	// ECS costs are primarily from the underlying infrastructure
	// For clusters, we estimate minimal overhead
	// For services, we estimate based on typical task requirements
	if resource.Type == "cluster" {
		estimate.Amount = 5.0
		estimate.Breakdown["management"] = 5.0
		estimate.Explanation = fmt.Sprintf("ECS cluster %s: $5.00/month", resource.Name)
	} else if resource.Type == "service" {
		estimate.Amount = 15.0
		estimate.Breakdown["tasks"] = 15.0
		estimate.Explanation = fmt.Sprintf("ECS service %s: $15.00/month", resource.Name)
		estimate.Assumptions = append(estimate.Assumptions, "Assumes moderate task requirements (0.5 vCPU, 1GB RAM)")
	}

	return estimate
}

// HTML template for the inventory report
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AWS Resource Inventory</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .header p {
            margin: 10px 0 0 0;
            opacity: 0.9;
        }
        .summary {
            padding: 30px;
            border-bottom: 1px solid #eee;
        }
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 6px;
            text-align: center;
        }
        .summary-card h3 {
            margin: 0 0 10px 0;
            color: #495057;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .summary-card .value {
            font-size: 2em;
            font-weight: bold;
            color: #212529;
        }
        .cost-estimates {
            background: #e8f5e8;
            padding: 20px;
            border-radius: 6px;
            margin-top: 20px;
        }
        .cost-estimates h3 {
            margin: 0 0 15px 0;
            color: #2d5a2d;
        }
        .cost-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 15px;
        }
        .cost-item {
            background: white;
            padding: 15px;
            border-radius: 4px;
            text-align: center;
        }
        .cost-item .service {
            font-weight: bold;
            color: #495057;
            text-transform: uppercase;
            font-size: 0.8em;
        }
        .cost-item .amount {
            font-size: 1.5em;
            font-weight: bold;
            color: #28a745;
        }
        .cost-item .explanation {
            font-size: 0.9em;
            color: #6c757d;
            margin-top: 8px;
            line-height: 1.4;
        }
        .cost-item .assumptions {
            margin-top: 12px;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
            font-size: 0.8em;
        }
        .cost-item .assumptions ul {
            margin: 5px 0 0 0;
            padding-left: 15px;
        }
        .cost-item .assumptions li {
            margin-bottom: 3px;
        }
        
        /* New Cost Breakdown Styles */
        .cost-summary {
            background: #fff;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            text-align: center;
        }
        .total-cost {
            font-size: 1.5em;
            font-weight: bold;
        }
        .total-cost .label {
            color: #495057;
        }
        .total-cost .amount {
            color: #28a745;
            margin-left: 10px;
        }
        .cost-breakdown {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 20px;
        }
        .cost-service-card {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .service-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .service-name {
            font-size: 1.2em;
            font-weight: bold;
        }
        .service-amount {
            font-size: 1.5em;
            font-weight: bold;
        }
        .cost-details {
            padding: 20px;
        }
        .formula-section, .examples-section, .assumptions-section {
            margin-bottom: 20px;
        }
        .formula-section h4, .examples-section h4, .assumptions-section h4 {
            margin: 0 0 10px 0;
            color: #495057;
            font-size: 1em;
        }
        .formula {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            font-family: 'Courier New', monospace;
            font-weight: bold;
            color:rgb(228, 233, 237);
            margin-bottom: 10px;
        }
        .formula-explanation {
            color: #6c757d;
            font-size: 0.9em;
            line-height: 1.4;
        }
        .examples-list, .assumptions-list {
            margin: 0;
            padding-left: 20px;
        }
        .examples-list li, .assumptions-list li {
            margin-bottom: 8px;
            line-height: 1.4;
        }
        .examples-list li {
            color: #495057;
        }
        .assumptions-list li {
            color: #dc3545;
        }
        
        /* Cost Breakdown by Type Styles */
        .cost-breakdown-by-type {
            margin-top: 20px;
            padding: 20px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .cost-breakdown-by-type h4 {
            margin: 0 0 15px 0;
            color: #495057;
        }
        .cost-type-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }
        .cost-type-card {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            text-align: center;
            border-left: 4px solid #28a745;
        }
        .cost-type-card .type-name {
            font-weight: bold;
            color: #495057;
            font-size: 1.1em;
            margin-bottom: 5px;
        }
        .cost-type-card .type-amount {
            font-size: 1.3em;
            font-weight: bold;
            color: #28a745;
            margin-bottom: 5px;
        }
        .cost-type-card .type-service {
            font-size: 0.8em;
            color: #6c757d;
            text-transform: uppercase;
        }
        
        /* Collapsible Resources Styles */
        .resources {
            padding: 30px;
        }
        .resources-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }
        .resources-header h2 {
            margin: 0;
        }
        .resource-controls {
            display: flex;
            gap: 10px;
        }
        .btn {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.9em;
            font-weight: 500;
        }
        .btn-primary {
            background: #007bff;
            color: white;
        }
        .btn-secondary {
            background: #6c757d;
            color: white;
        }
        .btn:hover {
            opacity: 0.8;
        }
        .resource-groups {
            display: flex;
            flex-direction: column;
            gap: 15px;
        }
        .resource-group {
            border: 1px solid #dee2e6;
            border-radius: 8px;
            overflow: hidden;
        }
        .group-header {
            background: #f8f9fa;
            padding: 15px 20px;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
            transition: background-color 0.2s;
        }
        .group-header:hover {
            background: #e9ecef;
        }
        .group-title {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .resource-count {
            color: #6c757d;
            font-size: 0.9em;
        }
        .group-toggle {
            font-size: 1.2em;
            transition: transform 0.2s;
        }
        .group-content {
            display: block;
            max-height: none;
            overflow: hidden;
            transition: max-height 0.3s ease;
        }
        .group-content.collapsed {
            max-height: 0;
        }
        .group-header.collapsed .group-toggle {
            transform: rotate(-90deg);
        }
        
        /* Cost Cell Styles */
        .cost-cell {
            cursor: pointer;
            color: #28a745;
            font-weight: bold;
            position: relative;
        }
        .cost-cell:hover {
            text-decoration: underline;
        }
        
        /* Tooltip Styles */
        .cost-tooltip {
            position: fixed;
            background: #333;
            color: white;
            padding: 15px;
            border-radius: 6px;
            font-size: 0.9em;
            max-width: 400px;
            z-index: 1000;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            display: none;
            user-select: text; /* Allow text selection */
            cursor: text;
            line-height: 1.4;
            border: 1px solid #555;
        }
        .cost-tooltip h4 {
            margin: 0 0 10px 0;
            color: #fff;
            font-size: 1em;
        }
        .cost-tooltip .formula {
            background: #555;
            padding: 8px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            margin-bottom: 8px;
            user-select: text;
            cursor: text;
            word-break: break-word;
        }
        .cost-tooltip .explanation {
            margin-bottom: 8px;
            line-height: 1.4;
            user-select: text;
            cursor: text;
        }
        .cost-tooltip .examples {
            margin-bottom: 8px;
        }
        .cost-tooltip .examples h5 {
            margin: 0 0 5px 0;
            color: #ffd700;
            user-select: text;
            cursor: text;
        }
        .cost-tooltip .examples ul {
            margin: 0;
            padding-left: 15px;
        }
        .cost-tooltip .examples li {
            user-select: text;
            cursor: text;
            margin-bottom: 3px;
        }
        .cost-tooltip .assumptions {
            margin-top: 8px;
        }
        .cost-tooltip .assumptions h5 {
            margin: 0 0 5px 0;
            color: #ff6b6b;
            user-select: text;
            cursor: text;
        }
        .cost-tooltip .assumptions ul {
            margin: 0;
            padding-left: 15px;
        }
        .cost-tooltip .assumptions li {
            user-select: text;
            cursor: text;
            margin-bottom: 3px;
        }
        
        /* Sortable Table Styles */
        .resource-table {
            width: 100%;
            overflow-x: auto;
            overflow-y: visible;
            display: block;
            background: white;
            border-radius: 6px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            max-height: none;
            -webkit-overflow-scrolling: touch; /* smooth scrolling on mobile */
        }
        .resource-table::-webkit-scrollbar {
            height: 8px;
        }
        .resource-table::-webkit-scrollbar-track {
            background: #f1f1f1;
            border-radius: 4px;
        }
        .resource-table::-webkit-scrollbar-thumb {
            background: #c1c1c1;
            border-radius: 4px;
        }
        .resource-table::-webkit-scrollbar-thumb:hover {
            background: #a8a8a8;
        }
        .resource-table table {
            min-width: 1200px;
            width: 100%;
            border-collapse: collapse;
            display: table;
        }
        .resource-table th {
            cursor: pointer;
            user-select: none;
            position: relative;
            white-space: nowrap;
            min-width: 120px;
            background: #f8f9fa;
            padding: 15px;
            text-align: left;
            font-weight: 600;
            color: #495057;
            border-bottom: 1px solid #dee2e6;
        }
        .resource-table th:hover {
            background: #e9ecef;
        }
        .resource-table th::after {
            content: '↕';
            position: absolute;
            right: 8px;
            opacity: 0.5;
        }
        .resource-table th.sort-asc::after {
            content: '↑';
            opacity: 1;
        }
        .resource-table th.sort-desc::after {
            content: '↓';
            opacity: 1;
        }
        .resource-table td {
            white-space: nowrap;
            max-width: 250px;
            overflow: hidden;
            text-overflow: ellipsis;
            padding: 12px 15px;
            border-bottom: 1px solid #f8f9fa;
        }
        .resource-table tbody tr:hover {
            background: #f8f9fa;
        }
        .table-scroll-hint {
            position: absolute;
            right: 10px;
            top: 50%;
            transform: translateY(-50%);
            background: rgba(0,0,0,0.7);
            color: white;
            padding: 5px 10px;
            border-radius: 4px;
            font-size: 12px;
            pointer-events: none;
            opacity: 0;
            transition: opacity 0.3s;
        }
        .resource-table:hover .table-scroll-hint {
            opacity: 1;
        }
        
        .resources h2 {
            margin: 0 0 20px 0;
            color: #495057;
        }
        .service-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.75em;
            font-weight: bold;
            text-transform: uppercase;
        }
        .service-ec2 { background: #e3f2fd; color: #1976d2; }
        .service-rds { background: #f3e5f5; color: #7b1fa2; }
        .service-lambda { background: #fff3e0; color: #f57c00; }
        .service-s3 { background: #e8f5e8; color: #388e3c; }
        .service-dynamodb { background: #fff8e1; color: #fbc02d; }
        .service-sfn { background: #fce4ec; color: #c2185b; }
        .service-cloudwatch { background: #e0f2f1; color: #00796b; }
        .state-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 12px;
            font-size: 0.75em;
            font-weight: bold;
        }
        .state-running, .state-available { background: #d4edda; color: #155724; }
        .state-stopped, .state-stopping { background: #f8d7da; color: #721c24; }
        .state-pending { background: #fff3cd; color: #856404; }
        .errors {
            background: #f8d7da;
            color: #721c24;
            padding: 20px;
            border-radius: 6px;
            margin: 20px 0;
        }
        .errors h3 {
            margin: 0 0 15px 0;
        }
        .errors ul {
            margin: 0;
            padding-left: 20px;
        }
        .footer {
            background: #f8f9fa;
            padding: 20px 30px;
            text-align: center;
            color: #6c757d;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>AWS Resource Inventory</h1>
            <p>Generated on {{.GeneratedAt.Format "January 2, 2006 at 3:04 PM MST"}}</p>
        </div>

        <div class="summary">
            <div class="summary-grid">
                <div class="summary-card">
                    <h3>Total Resources</h3>
                    <div class="value">{{len .Resources}}</div>
                </div>
                <div class="summary-card">
                    <h3>Services</h3>
                    <div class="value">{{len .Summary.Services}}</div>
                </div>
                <div class="summary-card">
                    <h3>Regions</h3>
                    <div class="value">{{len .Summary.Regions}}</div>
                </div>
                <div class="summary-card">
                    <h3>Duration</h3>
                    <div class="value">{{.Summary.Duration}}</div>
                </div>
            </div>

            {{if .CostEstimates}}
            <div class="cost-estimates">
                <h3>💰 Cost Analysis & Estimates</h3>
                <div class="cost-summary">
                    <div class="total-cost">
                        <span class="label">Total Estimated Monthly Cost:</span>
                        <span class="amount">${{$total := 0.0}}{{range $service, $estimate := .CostEstimates}}{{$total = add $total $estimate.Amount}}{{end}}{{printf "%.2f" $total}}</span>
                    </div>
                </div>
                
                <div class="cost-breakdown-by-type">
                    <h4>📊 Cost Breakdown by Resource Type</h4>
                    <div class="cost-type-grid">
                        {{$typeCosts := makeSlice}}
                        {{range $.Resources}}
                            {{if .CostEstimate}}
                                {{$typeCosts = append $typeCosts (dict "Type" .Type "Amount" .CostEstimate.Amount "Service" .Service)}}
                            {{end}}
                        {{end}}
                        
                        {{$groupedCosts := makeSlice}}
                        {{range $typeCosts}}
                            {{$found := false}}
                            {{$currentType := .Type}}
                            {{$currentService := .Service}}
                            {{range $groupedCosts}}
                                {{if eq .Type $currentType}}
                                    {{$found = true}}
                                {{end}}
                            {{end}}
                            {{if not $found}}
                                {{$total := 0.0}}
                                {{range $typeCosts}}
                                    {{if eq .Type $currentType}}
                                        {{$total = add $total .Amount}}
                                    {{end}}
                                {{end}}
                                {{$groupedCosts = append $groupedCosts (dict "Type" $currentType "Amount" $total "Service" $currentService)}}
                            {{end}}
                        {{end}}
                        
                        {{range $groupedCosts}}
                        <div class="cost-type-card">
                            <div class="type-name">{{.Type}}</div>
                            <div class="type-amount">${{printf "%.2f" .Amount}}</div>
                            <div class="type-service">{{.Service}}</div>
                        </div>
                        {{end}}
                    </div>
                </div>
            </div>
            {{end}}

            {{if .Errors}}
            <div class="errors">
                <h3>Errors ({{len .Errors}})</h3>
                <ul>
                    {{range .Errors}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
            </div>
            {{end}}
        </div>

        {{if .Resources}}
        <div class="resources">
            <div class="resources-header">
                <h2>📦 Resources Inventory ({{len .Resources}})</h2>
                <div class="resource-controls">
                    <button class="btn btn-primary" onclick="expandAll()">Expand All</button>
                    <button class="btn btn-secondary" onclick="collapseAll()">Collapse All</button>
                </div>
            </div>
            
            <div class="resource-groups">
                {{$services := makeSlice}}{{range .Resources}}{{$services = append $services .Service}}{{end}}{{$uniqueServices := unique $services}}
                {{range $service := $uniqueServices}}
                <div class="resource-group">
                    <div class="group-header" onclick="toggleGroup('{{$service}}')">
                        <div class="group-title">
                            <span class="service-badge service-{{$service}}">{{$service | upper}}</span>
                            <span class="resource-count">{{$count := 0}}{{range $.Resources}}{{if eq .Service $service}}{{$count = addInt $count 1}}{{end}}{{end}}({{$count}} resources)</span>
                        </div>
                        <div class="group-toggle">▼</div>
                    </div>
                    <div class="group-content" id="group-{{$service}}">
                        <div class="resource-table" style="position: relative;">
                            <div class="table-scroll-hint">← Scroll to see more columns →</div>
                            <table>
                                <thead>
                                    <tr>
                                        <th>Region</th>
                                        <th>ID</th>
                                        <th>Name</th>
                                        <th>Type</th>
                                        <th>State</th>
                                        <th>Class</th>
                                        <th>Created</th>
                                        <th>Monthly Cost</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {{range $.Resources}}
                                    {{if eq .Service $service}}
                                    <tr>
                                        <td>{{.Region}}</td>
                                        <td>{{.ID}}</td>
                                        <td>{{.Name}}</td>
                                        <td>{{.Type}}</td>
                                        <td><span class="state-badge state-{{.State}}">{{.State}}</span></td>
                                        <td>{{.Class}}</td>
                                        <td>{{if .CreatedAt}}{{.CreatedAt.Format "2006-01-02"}}{{else}}-{{end}}</td>
                                        <td>
                                            {{if .CostEstimate}}
                                            <span class="cost-cell" 
                                                  data-formula="{{.CostEstimate.Formula}}"
                                                  data-explanation="{{.CostEstimate.FormulaExplanation}}"
                                                  data-examples="{{range .CostEstimate.Examples}}{{.}}|{{end}}"
                                                  data-assumptions="{{range .CostEstimate.Assumptions}}{{.}}|{{end}}">
                                                ${{printf "%.2f" .CostEstimate.Amount}}
                                            </span>
                                            {{else}}
                                            -
                                            {{end}}
                                        </td>
                                    </tr>
                                    {{end}}
                                    {{end}}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        <div class="footer">
            <p>Generated by awsinv - AWS Resource Inventory Tool</p>
        </div>
    </div>
    
    <script>
        // Collapsible resource groups functionality
        function toggleGroup(serviceName) {
            const content = document.getElementById('group-' + serviceName);
            const header = content.previousElementSibling;
            
            if (content.classList.contains('collapsed')) {
                content.classList.remove('collapsed');
                header.classList.remove('collapsed');
            } else {
                content.classList.add('collapsed');
                header.classList.add('collapsed');
            }
        }
        
        function expandAll() {
            const contents = document.querySelectorAll('.group-content');
            const headers = document.querySelectorAll('.group-header');
            
            contents.forEach(content => content.classList.remove('collapsed'));
            headers.forEach(header => header.classList.remove('collapsed'));
        }
        
        function collapseAll() {
            const contents = document.querySelectorAll('.group-content');
            const headers = document.querySelectorAll('.group-header');
            
            contents.forEach(content => content.classList.add('collapsed'));
            headers.forEach(header => header.classList.add('collapsed'));
        }
        
        // Cost tooltip functionality
        function showCostTooltip(event, element) {
            // Remove any existing tooltips first
            document.querySelectorAll('.cost-tooltip').forEach(t => t.remove());
            
            const tooltip = document.createElement('div');
            tooltip.className = 'cost-tooltip';
            
            // Get data and decode HTML entities
            function decodeHtml(html) {
                const txt = document.createElement('textarea');
                txt.innerHTML = html;
                return txt.value;
            }
            
            const formula = decodeHtml(element.getAttribute('data-formula') || 'No formula available');
            const explanation = decodeHtml(element.getAttribute('data-explanation') || 'No explanation available');
            const examplesStr = decodeHtml(element.getAttribute('data-examples') || '');
            const assumptionsStr = decodeHtml(element.getAttribute('data-assumptions') || '');
            
            const examples = examplesStr.split('|').filter(e => e.trim());
            const assumptions = assumptionsStr.split('|').filter(a => a.trim());
            
            tooltip.innerHTML = 
                '<h4>💰 Cost Breakdown</h4>' +
                '<div class="formula">' + formula + '</div>' +
                '<div class="explanation">' + explanation + '</div>' +
                (examples.length > 0 ? 
                    '<div class="examples">' +
                        '<h5>📝 Examples:</h5>' +
                        '<ul>' + examples.map(function(ex) { return '<li>' + ex + '</li>'; }).join('') + '</ul>' +
                    '</div>' : '') +
                (assumptions.length > 0 ? 
                    '<div class="assumptions">' +
                        '<h5>⚠️ Assumptions:</h5>' +
                        '<ul>' + assumptions.map(function(ass) { return '<li>' + ass + '</li>'; }).join('') + '</ul>' +
                    '</div>' : '');
            
            // First add to body to get measurements
            document.body.appendChild(tooltip);
            tooltip.style.display = 'block';
            tooltip.style.visibility = 'hidden'; // Hide while positioning
            
            // Get actual tooltip dimensions after rendering
            const tooltipRect = tooltip.getBoundingClientRect();
            const tooltipWidth = tooltipRect.width;
            const tooltipHeight = tooltipRect.height;
            
            // Simple positioning: just use mouse position
            let left = event.clientX + 10; // 10px right of cursor
            let top = event.clientY - 10;  // 10px above cursor
            
            console.log('Mouse position:', event.clientX, event.clientY); // Debug
            console.log('Setting tooltip position to:', left, top); // Debug
            
            // Apply position and make visible
            tooltip.style.position = 'fixed'; // Ensure fixed positioning
            tooltip.style.left = left + 'px';
            tooltip.style.top = top + 'px';
            tooltip.style.visibility = 'visible';
            
            // Debug: Check actual position after setting
            setTimeout(() => {
                const actualRect = tooltip.getBoundingClientRect();
                console.log('Actual tooltip position:', actualRect.left, actualRect.top);
                console.log('Expected position:', left, top);
                console.log('Difference:', actualRect.left - left, actualRect.top - top);
            }, 10);
            
            // Simple mouse tracking for tooltip interaction
            let isOverTooltipArea = false;
            
            const checkMousePosition = (e) => {
                const mouseX = e.clientX;
                const mouseY = e.clientY;
                const elementRect = element.getBoundingClientRect();
                const tooltipRect = tooltip.getBoundingClientRect();
                
                // Check if mouse is over the original element or the tooltip
                const overElement = mouseX >= elementRect.left && mouseX <= elementRect.right && 
                                  mouseY >= elementRect.top && mouseY <= elementRect.bottom;
                                  
                const overTooltip = mouseX >= tooltipRect.left && mouseX <= tooltipRect.right && 
                                  mouseY >= tooltipRect.top && mouseY <= tooltipRect.bottom;
                
                isOverTooltipArea = overElement || overTooltip;
                
                if (!isOverTooltipArea) {
                    // Add small delay before hiding to prevent accidental closes
                    setTimeout(() => {
                        if (!isOverTooltipArea) {
                            hideTooltip();
                        }
                    }, 300);
                }
            };
            
            const hideTooltip = () => {
                if (tooltip.parentNode) {
                    document.removeEventListener('mousemove', checkMousePosition);
                    tooltip.parentNode.removeChild(tooltip);
                }
            };
            
            // Start tracking mouse movement after a short delay
            setTimeout(() => {
                document.addEventListener('mousemove', checkMousePosition);
            }, 500); // Longer delay to give user time to move to tooltip
            
            // Auto-hide after 20 seconds (longer since user might want to select text)
            setTimeout(hideTooltip, 20000);
        }
        
        // Table sorting functionality
        function sortTable(table, columnIndex, type = 'string') {
            const tbody = table.querySelector('tbody');
            const rows = Array.from(tbody.querySelectorAll('tr'));
            
            rows.sort((a, b) => {
                let aVal = a.cells[columnIndex].textContent.trim();
                let bVal = b.cells[columnIndex].textContent.trim();
                
                if (type === 'number') {
                    // Handle cost cells - if it's "-", treat as 0
                    if (aVal === '-') aVal = '0';
                    if (bVal === '-') bVal = '0';
                    
                    // Extract numeric value from cost cells (remove $ and other non-numeric chars)
                    aVal = parseFloat(aVal.replace(/[$,]/g, '')) || 0;
                    bVal = parseFloat(bVal.replace(/[$,]/g, '')) || 0;
                } else if (type === 'date') {
                    if (aVal === '-') aVal = '1900-01-01';
                    if (bVal === '-') bVal = '1900-01-01';
                    aVal = new Date(aVal);
                    bVal = new Date(bVal);
                } else {
                    // String comparison
                    aVal = aVal.toLowerCase();
                    bVal = bVal.toLowerCase();
                }
                
                if (aVal < bVal) return -1;
                if (aVal > bVal) return 1;
                return 0;
            });
            
            // Clear existing rows
            rows.forEach(row => tbody.removeChild(row));
            
            // Add sorted rows
            rows.forEach(row => tbody.appendChild(row));
        }
        
        // Initialize with all groups expanded and add event listeners
        document.addEventListener('DOMContentLoaded', function() {
            // Add cost tooltip listeners with delegation for dynamically loaded content
            document.addEventListener('mouseover', function(e) {
                if (e.target.classList.contains('cost-cell')) {
                    console.log('Cost cell hovered at:', e.clientX, e.clientY); // Debug
                    showCostTooltip(e, e.target);
                }
            }, true);
            
            // Add sorting listeners with delegation for dynamically loaded content
            document.addEventListener('click', function(e) {
                if (e.target.tagName === 'TH' && e.target.closest('.resource-table')) {
                    const th = e.target;
                    const table = th.closest('table');
                    const headers = Array.from(table.querySelectorAll('th'));
                    const index = headers.indexOf(th);
                    const currentSort = th.getAttribute('data-sort') || 'none';
                    
                    console.log('Column clicked:', index, th.textContent); // Debug
                    
                    // Clear other sort indicators
                    headers.forEach(header => {
                        header.classList.remove('sort-asc', 'sort-desc');
                        header.setAttribute('data-sort', 'none');
                    });
                    
                    // Determine sort type and direction based on column content
                    let sortType = 'string';
                    const headerText = th.textContent.trim().toLowerCase();
                    if (headerText.includes('cost')) sortType = 'number';
                    else if (headerText.includes('created')) sortType = 'date';
                    
                    let newSort = 'asc';
                    if (currentSort === 'asc') {
                        newSort = 'desc';
                        th.classList.add('sort-desc');
                    } else {
                        th.classList.add('sort-asc');
                    }
                    
                    th.setAttribute('data-sort', newSort);
                    
                    console.log('Sorting:', sortType, newSort); // Debug
                    
                    // Sort the table
                    sortTable(table, index, sortType);
                    
                    // Reverse if descending
                    if (newSort === 'desc') {
                        const tbody = table.querySelector('tbody');
                        const rows = Array.from(tbody.querySelectorAll('tr'));
                        rows.reverse();
                        rows.forEach(row => tbody.appendChild(row));
                    }
                }
            });
        });
    </script>
</body>
</html>` 