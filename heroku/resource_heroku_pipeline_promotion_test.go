package heroku

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceHerokuPipelinePromotion_Schema(t *testing.T) {
	resource := resourceHerokuPipelinePromotion()

	// Test required fields
	requiredFields := []string{"pipeline", "source_app_id", "targets"}
	for _, field := range requiredFields {
		if _, ok := resource.Schema[field]; !ok {
			t.Errorf("Required field %s not found in schema", field)
		}
		if !resource.Schema[field].Required {
			t.Errorf("Field %s should be required", field)
		}
		if !resource.Schema[field].ForceNew {
			t.Errorf("Field %s should be ForceNew", field)
		}
	}

	// Test optional fields
	optionalFields := []string{"release_id"}
	for _, field := range optionalFields {
		if _, ok := resource.Schema[field]; !ok {
			t.Errorf("Optional field %s not found in schema", field)
		}
		if !resource.Schema[field].Optional {
			t.Errorf("Field %s should be optional", field)
		}
		if !resource.Schema[field].ForceNew {
			t.Errorf("Field %s should be ForceNew", field)
		}
	}

	// Test computed fields
	computedFields := []string{"status", "created_at", "promoted_release_id"}
	for _, field := range computedFields {
		if _, ok := resource.Schema[field]; !ok {
			t.Errorf("Computed field %s not found in schema", field)
		}
		if !resource.Schema[field].Computed {
			t.Errorf("Field %s should be computed", field)
		}
	}

	// Test targets field is a Set
	if resource.Schema["targets"].Type != schema.TypeSet {
		t.Errorf("targets field should be TypeSet")
	}
}

func TestResourceHerokuPipelinePromotion_ReleaseIdValidation(t *testing.T) {
	// This test validates that release_id parameter currently returns an error
	// Once Flow team adds API support, this test should be updated

	d := schema.TestResourceDataRaw(t, resourceHerokuPipelinePromotion().Schema, map[string]interface{}{
		"pipeline":      "01234567-89ab-cdef-0123-456789abcdef",
		"source_app_id": "01234567-89ab-cdef-0123-456789abcdef",
		"release_id":    "01234567-89ab-cdef-0123-456789abcdef",
		"targets":       []interface{}{"01234567-89ab-cdef-0123-456789abcdef"},
	})

	meta := &Config{} // Mock config

	err := resourceHerokuPipelinePromotionCreate(d, meta)
	if err == nil {
		t.Error("Expected error when release_id is provided, but got none")
	}

	expectedError := "release_id parameter"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedError, err.Error())
	}
}
