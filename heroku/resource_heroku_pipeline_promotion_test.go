package heroku

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceHerokuPipelinePromotion_Schema(t *testing.T) {
	resource := resourceHerokuPipelinePromotion()

	// Test required fields
	requiredFields := []string{"pipeline", "source_app_id", "targets", "release_id"}
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
