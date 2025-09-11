package heroku

import (
	"testing"
)

// Unit tests for peering connection generation support
func TestHerokuSpacePeeringConnectionAccepterGeneration(t *testing.T) {
	tests := []struct {
		name        string
		generation  string
		expectError bool
		description string
	}{
		{name: "Cedar generation should be supported", generation: "cedar", expectError: false, description: "Cedar supports peering connections"},
		{name: "Fir generation should be unsupported", generation: "fir", expectError: true, description: "Fir does not support peering connections"},
		{name: "Default generation (cedar) should be supported", generation: "", expectError: false, description: "Default cedar generation supports peering connections"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generation := tt.generation
			if generation == "" {
				generation = "cedar"
			}
			supported := IsFeatureSupported(generation, "space", "peering_connection")
			shouldError := !supported
			if shouldError != tt.expectError {
				t.Errorf("Expected error: %t, but got: %t for generation %s", tt.expectError, shouldError, generation)
			}
			t.Logf("✅ Generation: %s, Supported: %t, ShouldError: %t", generation, supported, shouldError)
		})
	}
}
