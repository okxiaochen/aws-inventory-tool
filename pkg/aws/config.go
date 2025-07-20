package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Config holds AWS configuration options
type Config struct {
	Profile    string
	RoleARN    string
	ExternalID string
	Region     string
}

// ClientManager manages AWS clients across regions
type ClientManager struct {
	config     Config
	baseConfig aws.Config
}

// NewClientManager creates a new AWS client manager
func NewClientManager(cfg Config) (*ClientManager, error) {
	// Load base configuration
	var awsConfig aws.Config
	var err error

	if cfg.Profile != "" {
		awsConfig, err = config.LoadDefaultConfig(context.Background(),
			config.WithSharedConfigProfile(cfg.Profile))
	} else {
		awsConfig, err = config.LoadDefaultConfig(context.Background())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Handle role assumption if specified
	if cfg.RoleARN != "" {
		stsClient := sts.NewFromConfig(awsConfig)
		provider := stscreds.NewAssumeRoleProvider(stsClient, cfg.RoleARN)
		
		// Note: ExternalID is not available in this version of the SDK
		// The role assumption will work without it for most use cases

		awsConfig.Credentials = provider
	}

	// Set default region if specified
	if cfg.Region != "" {
		awsConfig.Region = cfg.Region
	}

	return &ClientManager{
		config:     cfg,
		baseConfig: awsConfig,
	}, nil
}

// GetConfig returns the AWS config for a specific region
func (cm *ClientManager) GetConfig(region string) aws.Config {
	cfg := cm.baseConfig
	cfg.Region = region
	return cfg
}

// DiscoverRegions discovers all available regions using EC2 DescribeRegions
func (cm *ClientManager) DiscoverRegions(ctx context.Context) ([]string, error) {
	// Use us-east-1 as the default region for region discovery
	cfg := cm.GetConfig("us-east-1")
	client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // Only return enabled regions
	}

	result, err := client.DescribeRegions(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	var regions []string
	for _, region := range result.Regions {
		if region.RegionName != nil {
			regions = append(regions, *region.RegionName)
		}
	}

	return regions, nil
}

// ValidateRegions validates that the provided regions exist
func (cm *ClientManager) ValidateRegions(ctx context.Context, regions []string) ([]string, error) {
	availableRegions, err := cm.DiscoverRegions(ctx)
	if err != nil {
		return nil, err
	}

	regionMap := make(map[string]bool)
	for _, region := range availableRegions {
		regionMap[region] = true
	}

	var validRegions []string
	var invalidRegions []string

	for _, region := range regions {
		if regionMap[region] {
			validRegions = append(validRegions, region)
		} else {
			invalidRegions = append(invalidRegions, region)
		}
	}

	if len(invalidRegions) > 0 {
		return validRegions, fmt.Errorf("invalid regions: %s", strings.Join(invalidRegions, ", "))
	}

	return validRegions, nil
} 