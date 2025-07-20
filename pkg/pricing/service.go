package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"github.com/aws/aws-sdk-go-v2/service/support"
)

// PricingService handles AWS pricing API integration and caching
type PricingService struct {
	pricingClient *pricing.Client
	supportClient *support.Client
	cache         *PricingCache
	freeTier      *FreeTierService
	mu            sync.RWMutex
}

// PricingCache stores pricing data with TTL
type PricingCache struct {
	data map[string]CachedPrice
	mu   sync.RWMutex
}

// CachedPrice represents a cached pricing entry
type CachedPrice struct {
	Price     float64   `json:"price"`
	ExpiresAt time.Time `json:"expires_at"`
	Currency  string    `json:"currency"`
}

// FreeTierService handles free tier detection and calculations
type FreeTierService struct {
	accountAge   time.Duration
	isEligible   bool
	usage        map[string]FreeTierUsage
	mu           sync.RWMutex
}

// FreeTierUsage tracks free tier usage for a service
type FreeTierUsage struct {
	Service           string    `json:"service"`
	RemainingHours    float64   `json:"remaining_hours"`
	RemainingGB       float64   `json:"remaining_gb"`
	RemainingRequests int64     `json:"remaining_requests"`
	LastUpdated       time.Time `json:"last_updated"`
}

// PricingResult contains pricing information with free tier considerations
type PricingResult struct {
	HourlyPrice     float64
	MonthlyPrice    float64
	Currency        string
	FreeTierCovered bool
	FreeTierSavings float64
	Region          string
	Accuracy        string
	Source          string // "api", "cache", "fallback"
}

// ServiceConfig contains service-specific pricing configuration
type ServiceConfig struct {
	ServiceCode  string
	ProductFamily string
	AttributeFilters map[string]string
}

// NewPricingService creates a new pricing service instance
func NewPricingService(ctx context.Context) (*PricingService, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Pricing API is only available in us-east-1
	pricingClient := pricing.NewFromConfig(cfg)
	
	// Support API for account information
	supportClient := support.NewFromConfig(cfg)

	cache := &PricingCache{
		data: make(map[string]CachedPrice),
	}

	freeTier := &FreeTierService{
		usage: make(map[string]FreeTierUsage),
	}

	service := &PricingService{
		pricingClient: pricingClient,
		supportClient: supportClient,
		cache:         cache,
		freeTier:      freeTier,
	}

	// Initialize free tier information
	if err := service.initializeFreeTier(ctx); err != nil {
		log.Printf("Warning: Could not initialize free tier information: %v", err)
	}

	return service, nil
}

// initializeFreeTier initializes free tier eligibility and usage
func (ps *PricingService) initializeFreeTier(ctx context.Context) error {
	ps.freeTier.mu.Lock()
	defer ps.freeTier.mu.Unlock()

	// For now, assume account is eligible for free tier
	// In production, you would call Support API to get account creation date
	ps.freeTier.isEligible = true
	ps.freeTier.accountAge = time.Hour * 24 * 30 // Assume 30 days old

	// Initialize free tier usage for supported services
	ps.freeTier.usage["ec2"] = FreeTierUsage{
		Service:           "ec2",
		RemainingHours:    750.0, // 750 hours/month for t2.micro
		LastUpdated:       time.Now(),
	}

	ps.freeTier.usage["rds"] = FreeTierUsage{
		Service:           "rds",
		RemainingHours:    750.0, // 750 hours/month for db.t2.micro
		LastUpdated:       time.Now(),
	}

	ps.freeTier.usage["lambda"] = FreeTierUsage{
		Service:           "lambda",
		RemainingRequests: 1000000, // 1M requests/month
		LastUpdated:       time.Now(),
	}

	ps.freeTier.usage["s3"] = FreeTierUsage{
		Service:        "s3",
		RemainingGB:    5.0, // 5GB storage
		LastUpdated:    time.Now(),
	}

	ps.freeTier.usage["dynamodb"] = FreeTierUsage{
		Service:        "dynamodb",
		RemainingGB:    25.0, // 25GB storage
		LastUpdated:    time.Now(),
	}

	return nil
}

// GetServiceConfig returns pricing configuration for a service
func (ps *PricingService) GetServiceConfig(service string) ServiceConfig {
	configs := map[string]ServiceConfig{
		"ec2": {
			ServiceCode:   "AmazonEC2",
			ProductFamily: "Compute Instance",
			AttributeFilters: map[string]string{
				"tenancy":     "Shared",
				"capacitystatus": "Used",
				"preInstalledSw": "NA",
			},
		},
		"rds": {
			ServiceCode:   "AmazonRDS",
			ProductFamily: "Database Instance",
			AttributeFilters: map[string]string{
				"deploymentOption": "Single-AZ",
			},
		},
		"lambda": {
			ServiceCode:   "AWSLambda",
			ProductFamily: "Serverless",
		},
		"s3": {
			ServiceCode:   "AmazonS3",
			ProductFamily: "Storage",
		},
		"dynamodb": {
			ServiceCode:   "AmazonDynamoDB",
			ProductFamily: "Database Storage and IO",
		},
		"redis": {
			ServiceCode:   "AmazonElastiCache",
			ProductFamily: "Cache Instance",
		},
	}

	if config, exists := configs[service]; exists {
		return config
	}

	// Default config
	return ServiceConfig{
		ServiceCode:   "Unknown",
		ProductFamily: "Unknown",
	}
}

// GetPricing retrieves pricing for a specific resource
func (ps *PricingService) GetPricing(ctx context.Context, service, region, instanceType string) (*PricingResult, error) {
	cacheKey := fmt.Sprintf("%s-%s-%s", service, region, instanceType)

	// Check cache first
	if cachedPrice, found := ps.cache.get(cacheKey); found {
		freeTierResult := ps.checkFreeTier(service, instanceType, cachedPrice.Price)
		return &PricingResult{
			HourlyPrice:     cachedPrice.Price,
			MonthlyPrice:    cachedPrice.Price * 730, // 730 hours per month
			Currency:        cachedPrice.Currency,
			FreeTierCovered: freeTierResult.covered,
			FreeTierSavings: freeTierResult.savings,
			Region:          region,
			Accuracy:        "High",
			Source:          "cache",
		}, nil
	}

	// Get from AWS Pricing API
	serviceConfig := ps.GetServiceConfig(service)
	price, err := ps.fetchPricingFromAPI(ctx, serviceConfig, region, instanceType)
	if err != nil {
		log.Printf("Failed to fetch pricing from API for %s: %v", cacheKey, err)
		// Fall back to hardcoded estimates
		return ps.getFallbackPricing(service, region, instanceType), nil
	}

	// Cache the result
	ps.cache.set(cacheKey, CachedPrice{
		Price:     price,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Cache for 24 hours
		Currency:  "USD",
	})

	freeTierResult := ps.checkFreeTier(service, instanceType, price)
	return &PricingResult{
		HourlyPrice:     price,
		MonthlyPrice:    price * 730,
		Currency:        "USD",
		FreeTierCovered: freeTierResult.covered,
		FreeTierSavings: freeTierResult.savings,
		Region:          region,
		Accuracy:        "High",
		Source:          "api",
	}, nil
}

// fetchPricingFromAPI retrieves pricing from AWS Pricing API
func (ps *PricingService) fetchPricingFromAPI(ctx context.Context, serviceConfig ServiceConfig, region, instanceType string) (float64, error) {
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("ServiceCode"),
			Value: aws.String(serviceConfig.ServiceCode),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("location"),
			Value: aws.String(ps.getLocationFromRegion(region)),
		},
	}

	// Add instance type filter for compute services
	if instanceType != "" && (serviceConfig.ServiceCode == "AmazonEC2" || serviceConfig.ServiceCode == "AmazonRDS" || serviceConfig.ServiceCode == "AmazonElastiCache") {
		filters = append(filters, types.Filter{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("instanceType"),
			Value: aws.String(instanceType),
		})
	}

	// Add service-specific filters
	for field, value := range serviceConfig.AttributeFilters {
		filters = append(filters, types.Filter{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String(field),
			Value: aws.String(value),
		})
	}

	input := &pricing.GetProductsInput{
		ServiceCode:   aws.String(serviceConfig.ServiceCode),
		Filters:       filters,
		MaxResults:    aws.Int32(10),
	}

	resp, err := ps.pricingClient.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to get products: %w", err)
	}

	if len(resp.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing data found for %s %s in %s", serviceConfig.ServiceCode, instanceType, region)
	}

	// Parse the first result (pricing data is in JSON format)
	return ps.parsePricingData(resp.PriceList[0])
}

// parsePricingData extracts hourly price from AWS pricing JSON
func (ps *PricingService) parsePricingData(priceListItem string) (float64, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(priceListItem), &data); err != nil {
		return 0, fmt.Errorf("failed to parse pricing data: %w", err)
	}

	// Navigate through the complex JSON structure
	terms, ok := data["terms"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no terms found in pricing data")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no OnDemand terms found")
	}

	// Get the first (and usually only) on-demand term
	for _, termData := range onDemand {
		term, ok := termData.(map[string]interface{})
		if !ok {
			continue
		}

		priceDimensions, ok := term["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		// Get the first price dimension
		for _, dimData := range priceDimensions {
			dimension, ok := dimData.(map[string]interface{})
			if !ok {
				continue
			}

			pricePerUnit, ok := dimension["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}

			// Get USD price
			if usdPrice, exists := pricePerUnit["USD"]; exists {
				if priceStr, ok := usdPrice.(string); ok {
					return strconv.ParseFloat(priceStr, 64)
				}
			}
		}
	}

	return 0, fmt.Errorf("no valid price found in pricing data")
}

// getLocationFromRegion converts AWS region to location name used in pricing API
func (ps *PricingService) getLocationFromRegion(region string) string {
	locationMap := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "Europe (Ireland)",
		"eu-west-2":      "Europe (London)",
		"eu-west-3":      "Europe (Paris)",
		"eu-central-1":   "Europe (Frankfurt)",
		"eu-north-1":     "Europe (Stockholm)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-northeast-2": "Asia Pacific (Seoul)",
		"ap-northeast-3": "Asia Pacific (Osaka)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
		"ca-central-1":   "Canada (Central)",
		"sa-east-1":      "South America (São Paulo)",
	}

	if location, exists := locationMap[region]; exists {
		return location
	}
	return "US East (N. Virginia)" // Default fallback
}

// freeTierCheck represents free tier check result
type freeTierCheck struct {
	covered bool
	savings float64
}

// checkFreeTier determines if a resource is covered by free tier
func (ps *PricingService) checkFreeTier(service, instanceType string, hourlyPrice float64) freeTierCheck {
	ps.freeTier.mu.RLock()
	defer ps.freeTier.mu.RUnlock()

	if !ps.freeTier.isEligible {
		return freeTierCheck{covered: false, savings: 0}
	}

	usage, exists := ps.freeTier.usage[service]
	if !exists {
		return freeTierCheck{covered: false, savings: 0}
	}

	switch service {
	case "ec2":
		if instanceType == "t2.micro" && usage.RemainingHours > 0 {
			monthlyHours := 730.0
			coveredHours := usage.RemainingHours
			if coveredHours >= monthlyHours {
				// Fully covered by free tier
				return freeTierCheck{covered: true, savings: hourlyPrice * monthlyHours}
			} else if coveredHours > 0 {
				// Partially covered
				savings := hourlyPrice * coveredHours
				return freeTierCheck{covered: false, savings: savings}
			}
		}
	case "rds":
		if instanceType == "db.t2.micro" && usage.RemainingHours > 0 {
			monthlyHours := 730.0
			coveredHours := usage.RemainingHours
			if coveredHours >= monthlyHours {
				return freeTierCheck{covered: true, savings: hourlyPrice * monthlyHours}
			} else if coveredHours > 0 {
				savings := hourlyPrice * coveredHours
				return freeTierCheck{covered: false, savings: savings}
			}
		}
	case "lambda":
		if usage.RemainingRequests > 0 {
			// Lambda free tier is complex, for now assume partial coverage
			return freeTierCheck{covered: false, savings: 5.0} // Rough estimate
		}
	case "s3":
		if usage.RemainingGB > 0 {
			// S3 free tier provides 5GB
			return freeTierCheck{covered: false, savings: 1.0} // Rough estimate
		}
	case "dynamodb":
		if usage.RemainingGB > 0 {
			// DynamoDB free tier is generous
			return freeTierCheck{covered: true, savings: 10.0} // Rough estimate
		}
	}

	return freeTierCheck{covered: false, savings: 0}
}

// getFallbackPricing returns hardcoded estimates when API fails
func (ps *PricingService) getFallbackPricing(service, region, instanceType string) *PricingResult {
	// Fallback to our existing hardcoded estimates
	fallbackPrices := map[string]map[string]float64{
		"ec2": {
			"t3.micro":  0.0116,
			"t3.small":  0.0232,
			"t3.medium": 0.0464,
			"t3.large":  0.0928,
			"m5.large":  0.1184,
			"default":   0.05,
		},
		"rds": {
			"db.t3.micro":  0.0205,
			"db.t3.small":  0.041,
			"db.t3.medium": 0.082,
			"db.m5.large":  0.234,
			"default":      0.1,
		},
		"redis": {
			"cache.t3.micro":  0.017,
			"cache.t3.small":  0.034,
			"cache.m5.large":  0.136,
			"default":         0.05,
		},
	}

	var hourlyPrice float64
	if servicePrices, exists := fallbackPrices[service]; exists {
		if price, exists := servicePrices[instanceType]; exists {
			hourlyPrice = price
		} else {
			hourlyPrice = servicePrices["default"]
		}
	} else {
		hourlyPrice = 0.05 // Default fallback
	}

	freeTierResult := ps.checkFreeTier(service, instanceType, hourlyPrice)
	return &PricingResult{
		HourlyPrice:     hourlyPrice,
		MonthlyPrice:    hourlyPrice * 730,
		Currency:        "USD",
		FreeTierCovered: freeTierResult.covered,
		FreeTierSavings: freeTierResult.savings,
		Region:          region,
		Accuracy:        "Medium",
		Source:          "fallback",
	}
}

// Cache methods
func (cache *PricingCache) get(key string) (CachedPrice, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if price, exists := cache.data[key]; exists {
		if time.Now().Before(price.ExpiresAt) {
			return price, true
		}
		// Expired, remove it
		delete(cache.data, key)
	}
	return CachedPrice{}, false
}

func (cache *PricingCache) set(key string, price CachedPrice) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.data[key] = price
}

// GetFreeTierInfo returns current free tier information
func (ps *PricingService) GetFreeTierInfo() map[string]FreeTierUsage {
	ps.freeTier.mu.RLock()
	defer ps.freeTier.mu.RUnlock()

	result := make(map[string]FreeTierUsage)
	for service, usage := range ps.freeTier.usage {
		result[service] = usage
	}
	return result
}

// IsFreeTierEligible returns whether the account is eligible for free tier
func (ps *PricingService) IsFreeTierEligible() bool {
	ps.freeTier.mu.RLock()
	defer ps.freeTier.mu.RUnlock()
	return ps.freeTier.isEligible
} 