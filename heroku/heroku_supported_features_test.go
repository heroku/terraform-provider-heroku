package heroku

import (
	"testing"
)

func TestIsFeatureSupported(t *testing.T) {
	testCases := []struct {
		name         string
		generation   string
		resourceType string
		feature      string
		expected     bool
	}{
		// Cedar generation tests - all space features supported
		{
			name:         "Cedar space private should be supported",
			generation:   "cedar",
			resourceType: "space",
			feature:      "private",
			expected:     true,
		},
		{
			name:         "Cedar space shield should be supported",
			generation:   "cedar",
			resourceType: "space",
			feature:      "shield",
			expected:     true,
		},
		{
			name:         "Cedar space trusted_ip_ranges should be supported",
			generation:   "cedar",
			resourceType: "space",
			feature:      "trusted_ip_ranges",
			expected:     true,
		},

		// Fir generation tests - private supported, shield and others unsupported
		{
			name:         "Fir space private should be supported",
			generation:   "fir",
			resourceType: "space",
			feature:      "private",
			expected:     true,
		},
		{
			name:         "Fir space shield should be unsupported",
			generation:   "fir",
			resourceType: "space",
			feature:      "shield",
			expected:     false,
		},
		{
			name:         "Fir space trusted_ip_ranges should be unsupported",
			generation:   "fir",
			resourceType: "space",
			feature:      "trusted_ip_ranges",
			expected:     false,
		},

		// Unsupported feature tests (features not in matrix)
		{
			name:         "Cedar space unknown_feature should be unsupported (not in matrix)",
			generation:   "cedar",
			resourceType: "space",
			feature:      "unknown_feature",
			expected:     false,
		},
		{
			name:         "Fir space unknown_feature should be unsupported (not in matrix)",
			generation:   "fir",
			resourceType: "space",
			feature:      "unknown_feature",
			expected:     false,
		},

		// Unknown resource type tests
		// App feature tests
		{
			name:         "Cedar app buildpacks should be supported",
			generation:   "cedar",
			resourceType: "app",
			feature:      "buildpacks",
			expected:     true,
		},
		{
			name:         "Cedar app stack should be supported",
			generation:   "cedar",
			resourceType: "app",
			feature:      "stack",
			expected:     true,
		},
		{
			name:         "Cedar app internal_routing should be supported",
			generation:   "cedar",
			resourceType: "app",
			feature:      "internal_routing",
			expected:     true,
		},
		{
			name:         "Cedar app cloud_native_buildpacks should be unsupported",
			generation:   "cedar",
			resourceType: "app",
			feature:      "cloud_native_buildpacks",
			expected:     false,
		},
		{
			name:         "Fir app buildpacks should be unsupported",
			generation:   "fir",
			resourceType: "app",
			feature:      "buildpacks",
			expected:     false,
		},
		{
			name:         "Fir app stack should be unsupported",
			generation:   "fir",
			resourceType: "app",
			feature:      "stack",
			expected:     false,
		},
		{
			name:         "Fir app internal_routing should be unsupported",
			generation:   "fir",
			resourceType: "app",
			feature:      "internal_routing",
			expected:     false,
		},
		{
			name:         "Fir app cloud_native_buildpacks should be supported",
			generation:   "fir",
			resourceType: "app",
			feature:      "cloud_native_buildpacks",
			expected:     true,
		},
		{
			name:         "Fir build features should be unsupported (not implemented yet)",
			generation:   "fir",
			resourceType: "build",
			feature:      "some_feature",
			expected:     false,
		},

		// Invalid generation tests
		{
			name:         "Invalid generation should be unsupported",
			generation:   "invalid",
			resourceType: "space",
			feature:      "shield",
			expected:     false,
		},
		{
			name:         "Empty generation should be unsupported",
			generation:   "",
			resourceType: "space",
			feature:      "shield",
			expected:     false,
		},

		// Edge case tests
		{
			name:         "Empty resource type should be unsupported",
			generation:   "cedar",
			resourceType: "",
			feature:      "shield",
			expected:     false,
		},
		{
			name:         "Empty feature should be unsupported",
			generation:   "cedar",
			resourceType: "space",
			feature:      "",
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsFeatureSupported(tc.generation, tc.resourceType, tc.feature)
			if result != tc.expected {
				t.Errorf("IsFeatureSupported(%q, %q, %q) = %v, expected %v",
					tc.generation, tc.resourceType, tc.feature, result, tc.expected)
			}
		})
	}
}

// TestFeatureMatrixConsistency ensures the feature matrix is internally consistent
func TestFeatureMatrixConsistency(t *testing.T) {
	// Verify that all generations have at least one resource
	for generation, resources := range featureMatrix {
		if len(resources) == 0 {
			t.Errorf("Generation %s has no resources defined", generation)
		}

		// Verify that all resources have at least one feature
		for resourceType, features := range resources {
			if len(features) == 0 {
				t.Errorf("Generation %s, resource %s has no features defined", generation, resourceType)
			}

			// Verify that all features are properly set (no nil values)
			for feature, supported := range features {
				if feature == "" {
					t.Errorf("Generation %s, resource %s has empty feature name", generation, resourceType)
				}
				// supported is bool, so just verify it's not accidentally unset in a way that would matter
				_ = supported // This is intentional - we're just ensuring the value exists
			}
		}
	}

	// Verify minimum required features exist for Task 1
	// Both generations support private spaces
	if !IsFeatureSupported("cedar", "space", "private") {
		t.Error("Cedar space private must be supported")
	}
	if !IsFeatureSupported("fir", "space", "private") {
		t.Error("Fir space private must be supported")
	}

	// Only cedar supports shield spaces
	if !IsFeatureSupported("cedar", "space", "shield") {
		t.Error("Cedar space shield must be supported")
	}
	if IsFeatureSupported("fir", "space", "shield") {
		t.Error("Fir space shield must be unsupported")
	}
}
