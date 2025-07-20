package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestResource_JSON(t *testing.T) {
	now := time.Now()
	resource := Resource{
		Service:   "ec2",
		Region:    "us-east-1",
		ID:        "i-1234567890abcdef0",
		Name:      "test-instance",
		Type:      "t3.micro",
		State:     "running",
		Class:     "t3.micro",
		CreatedAt: &now,
		Tags: map[string]string{
			"Environment": "test",
			"Project":     "awsinv",
		},
		Extra: map[string]interface{}{
			"privateIp": "10.0.1.100",
		},
	}

	// Test that the resource can be marshaled to JSON
	jsonData, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource to JSON: %v", err)
	}

	// Test that the JSON can be unmarshaled back to a resource
	var unmarshaled Resource
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to resource: %v", err)
	}

	// Compare the original and unmarshaled resources
	if diff := cmp.Diff(resource, unmarshaled); diff != "" {
		t.Errorf("Resource mismatch after JSON round-trip (-want +got):\n%s", diff)
	}
}

func TestResourceCollection_Summary(t *testing.T) {
	collection := &ResourceCollection{
		Resources: []Resource{
			{Service: "ec2", Region: "us-east-1", State: "running"},
			{Service: "ec2", Region: "us-east-1", State: "stopped"},
			{Service: "rds", Region: "us-west-2", State: "available"},
		},
		Summary: Summary{
			TotalResources: 3,
			ByService: map[string]int{
				"ec2": 2,
				"rds": 1,
			},
			ByRegion: map[string]int{
				"us-east-1": 2,
				"us-west-2": 1,
			},
			ByState: map[string]int{
				"running":    1,
				"stopped":    1,
				"available":  1,
			},
		},
	}

	// Test that the summary is correct
	if collection.Summary.TotalResources != 3 {
		t.Errorf("Expected 3 total resources, got %d", collection.Summary.TotalResources)
	}

	if collection.Summary.ByService["ec2"] != 2 {
		t.Errorf("Expected 2 EC2 resources, got %d", collection.Summary.ByService["ec2"])
	}

	if collection.Summary.ByRegion["us-east-1"] != 2 {
		t.Errorf("Expected 2 resources in us-east-1, got %d", collection.Summary.ByRegion["us-east-1"])
	}
} 