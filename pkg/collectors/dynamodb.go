package collectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awspkg "github.com/xiaochen/awsinv/pkg/aws"
	"github.com/xiaochen/awsinv/pkg/models"
)

// DynamoDBCollector collects DynamoDB tables
type DynamoDBCollector struct {
	clientManager *awspkg.ClientManager
}

// NewDynamoDBCollector creates a new DynamoDB collector
func NewDynamoDBCollector(clientManager *awspkg.ClientManager) *DynamoDBCollector {
	return &DynamoDBCollector{
		clientManager: clientManager,
	}
}

// Name returns the service name
func (c *DynamoDBCollector) Name() string {
	return "dynamodb"
}

// Regions returns the regions this collector supports
func (c *DynamoDBCollector) Regions() []string {
	// DynamoDB is available in all regions
	return nil // Will be populated by the orchestrator
}

// Collect retrieves DynamoDB tables for the given region
func (c *DynamoDBCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
	cfg := c.clientManager.GetConfig(region)
	client := dynamodb.NewFromConfig(cfg)

	var resources []models.Resource
	var lastEvaluatedTableName *string

	for {
		input := &dynamodb.ListTablesInput{
			ExclusiveStartTableName: lastEvaluatedTableName,
		}

		result, err := client.ListTables(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list tables in %s: %w", region, err)
		}

		// Get detailed information for each table
		for _, tableName := range result.TableNames {
			tableInfo, err := c.getTableInfo(ctx, client, tableName)
			if err != nil {
				// Log error but continue with other tables
				fmt.Printf("Warning: failed to get info for table %s: %v\n", tableName, err)
				continue
			}
			resource := c.convertTable(tableInfo, region)
			resources = append(resources, resource)
		}

		lastEvaluatedTableName = result.LastEvaluatedTableName
		if lastEvaluatedTableName == nil {
			break
		}
	}

	return resources, nil
}

// getTableInfo retrieves detailed information about a DynamoDB table
func (c *DynamoDBCollector) getTableInfo(ctx context.Context, client *dynamodb.Client, tableName string) (*types.TableDescription, error) {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	result, err := client.DescribeTable(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.Table, nil
}

// convertTable converts a DynamoDB table to a Resource
func (c *DynamoDBCollector) convertTable(table *types.TableDescription, region string) models.Resource {
	resource := models.Resource{
		Service: "dynamodb",
		Region:  region,
		ID:      aws.ToString(table.TableName),
		Name:    aws.ToString(table.TableName),
		Type:    "table",
		State:   string(table.TableStatus),
		Class:   "table",
	}

	// Set creation time
	if table.CreationDateTime != nil {
		createdAt := aws.ToTime(table.CreationDateTime)
		resource.CreatedAt = &createdAt
	}

	// Add extra information
	extra := make(map[string]interface{})
	if table.TableArn != nil {
		extra["tableArn"] = aws.ToString(table.TableArn)
	}
	if table.TableId != nil {
		extra["tableId"] = aws.ToString(table.TableId)
	}
	if table.ItemCount != nil {
		extra["itemCount"] = aws.ToInt64(table.ItemCount)
	}
	if table.TableSizeBytes != nil {
		extra["tableSizeBytes"] = aws.ToInt64(table.TableSizeBytes)
	}
	// Note: BillingMode is not available in this version of the SDK
	// extra["billingMode"] = "unknown"
	if table.ProvisionedThroughput != nil {
		extra["readCapacityUnits"] = aws.ToInt64(table.ProvisionedThroughput.ReadCapacityUnits)
		extra["writeCapacityUnits"] = aws.ToInt64(table.ProvisionedThroughput.WriteCapacityUnits)
	}
	if table.GlobalSecondaryIndexes != nil {
		extra["globalSecondaryIndexes"] = len(table.GlobalSecondaryIndexes)
	}
	if table.LocalSecondaryIndexes != nil {
		extra["localSecondaryIndexes"] = len(table.LocalSecondaryIndexes)
	}
	if table.StreamSpecification != nil && table.StreamSpecification.StreamEnabled != nil {
		extra["streamEnabled"] = aws.ToBool(table.StreamSpecification.StreamEnabled)
	}
	if table.SSEDescription != nil {
		extra["encryptionType"] = string(table.SSEDescription.SSEType)
	}

	resource.Extra = extra

	return resource
} 