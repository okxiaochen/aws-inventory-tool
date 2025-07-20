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

	// Print summary
	fmt.Fprintf(f.writer, "\nAWS Resource Inventory Summary\n")
	fmt.Fprintf(f.writer, "==============================\n")
	fmt.Fprintf(f.writer, "Total Resources: %d\n", len(resources))
	fmt.Fprintf(f.writer, "Duration: %v\n", collection.Summary.Duration)
	fmt.Fprintf(f.writer, "Errors: %d\n", len(collection.Errors))

	if len(collection.Summary.ByService) > 0 {
		fmt.Fprintf(f.writer, "\nBy Service:\n")
		for service, count := range collection.Summary.ByService {
			fmt.Fprintf(f.writer, "  %s: %d\n", service, count)
		}
	}

	if len(collection.Summary.ByRegion) > 0 {
		fmt.Fprintf(f.writer, "\nBy Region:\n")
		for region, count := range collection.Summary.ByRegion {
			fmt.Fprintf(f.writer, "  %s: %d\n", region, count)
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
		fmt.Fprintf(f.writer, "\nResources:\n")
		fmt.Fprintf(f.writer, "%-12s %-15s %-20s %-15s %-10s %-10s %-10s\n", "SERVICE", "REGION", "ID", "NAME", "TYPE", "STATE", "CLASS")
		fmt.Fprintf(f.writer, "%-12s %-15s %-20s %-15s %-10s %-10s %-10s\n", "-------", "------", "--", "----", "----", "-----", "-----")

		for _, resource := range resources {
			fmt.Fprintf(f.writer, "%-12s %-15s %-20s %-15s %-10s %-10s %-10s\n",
				truncate(resource.Service, 12),
				truncate(resource.Region, 15),
				truncate(resource.ID, 20),
				truncate(resource.Name, 15),
				truncate(resource.Type, 10),
				truncate(resource.State, 10),
				truncate(resource.Class, 10))
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

// Format formats the collection as JSON
func (f *JSONFormatter) Format(collection *models.ResourceCollection, filters []Filter, sortField string, noColor bool) error {
	// Apply filters
	resources := applyFilters(collection.Resources, filters)

	// Sort resources
	sortResources(resources, sortField)

	// Create output structure
	output := struct {
		Resources []models.Resource `json:"resources"`
		Summary   models.Summary    `json:"summary"`
		Errors    []string          `json:"errors,omitempty"`
	}{
		Resources: resources,
		Summary:   collection.Summary,
		Errors:    collection.Errors,
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

	writer := csv.NewWriter(f.writer)
	defer writer.Flush()

	// Write header
	header := []string{"Service", "Region", "ID", "Name", "Type", "State", "Class", "CreatedAt", "Tags"}
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

		row := []string{
			resource.Service,
			resource.Region,
			resource.ID,
			resource.Name,
			resource.Type,
			resource.State,
			resource.Class,
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