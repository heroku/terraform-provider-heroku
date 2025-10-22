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
	computedFields := []string{"status", "created_at", "promoted_release_id", "promoted_release_ids"}
	for _, field := range computedFields {
		if _, ok := resource.Schema[field]; !ok {
			t.Errorf("Computed field %s not found in schema", field)
		}
		if !resource.Schema[field].Computed {
			t.Errorf("Field %s should be computed", field)
		}
	}

	// Test promoted_release_id is deprecated
	if resource.Schema["promoted_release_id"].Deprecated == "" {
		t.Errorf("promoted_release_id should be marked as deprecated")
	}

	// Test promoted_release_ids is a List of objects
	if resource.Schema["promoted_release_ids"].Type != schema.TypeList {
		t.Errorf("promoted_release_ids field should be TypeList")
	}

	// Test promoted_release_ids contains objects with app_id and release_id
	promotedReleaseIdsElem, ok := resource.Schema["promoted_release_ids"].Elem.(*schema.Resource)
	if !ok {
		t.Fatal("Expected promoted_release_ids.Elem to be a Resource (object)")
	}

	if _, hasAppID := promotedReleaseIdsElem.Schema["app_id"]; !hasAppID {
		t.Fatal("Expected promoted_release_ids objects to have app_id field")
	}

	if _, hasReleaseID := promotedReleaseIdsElem.Schema["release_id"]; !hasReleaseID {
		t.Fatal("Expected promoted_release_ids objects to have release_id field")
	}

	// Test targets field is a Set
	if resource.Schema["targets"].Type != schema.TypeSet {
		t.Errorf("targets field should be TypeSet")
	}
}
