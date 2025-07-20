package models

import (
	"context"
	"time"
)

// Resource represents a normalized AWS resource across all services
type Resource struct {
	Service      string                 `json:"service"`
	Region       string                 `json:"region"`
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	Type         string                 `json:"type,omitempty"`          // instance type, engine, runtime...
	State        string                 `json:"state,omitempty"`
	Class        string                 `json:"class,omitempty"`         // db class, memory size, etc.
	CreatedAt    *time.Time             `json:"createdAt,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// ResourceCollection represents a collection of resources with metadata
type ResourceCollection struct {
	Resources []Resource `json:"resources"`
	Errors    []string   `json:"errors,omitempty"`
	Summary   Summary    `json:"summary"`
}

// Summary provides statistics about the inventory
type Summary struct {
	TotalResources int                    `json:"totalResources"`
	ByService      map[string]int         `json:"byService"`
	ByRegion       map[string]int         `json:"byRegion"`
	ByState        map[string]int         `json:"byState"`
	Errors         int                    `json:"errors"`
	Duration       time.Duration          `json:"duration"`
	Regions        []string               `json:"regions"`
	Services       []string               `json:"services"`
}

// Collector defines the interface for AWS service collectors
type Collector interface {
	// Name returns the service name (e.g., "ec2", "rds")
	Name() string
	
	// Collect retrieves resources for the given region
	Collect(ctx context.Context, region string) ([]Resource, error)
	
	// Regions returns the list of regions this collector supports
	Regions() []string
}

// CollectorResult represents the result of a collector operation
type CollectorResult struct {
	Service   string
	Region    string
	Resources []Resource
	Error     error
} 