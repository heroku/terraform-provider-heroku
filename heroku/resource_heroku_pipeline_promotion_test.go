package heroku

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestResourceHerokuPipelinePromotion_Schema(t *testing.T) {
	r, ok := NewPipelinePromotionResource().(resource.ResourceWithConfigure)
	if !ok {
		t.Fatal("expected pipeline_promotion resource to implement resource.ResourceWithConfigure")
	}

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	s := resp.Schema

	// Required + ForceNew (RequiresReplace) fields.
	requiredFields := []string{"pipeline", "source_app_id", "targets", "release_id"}
	for _, field := range requiredFields {
		attr, ok := s.Attributes[field]
		if !ok {
			t.Errorf("Required field %s not found in schema", field)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("Field %s should be required", field)
		}
	}

	// Computed fields.
	computedFields := []string{"status", "created_at", "promoted_release_id"}
	for _, field := range computedFields {
		attr, ok := s.Attributes[field]
		if !ok {
			t.Errorf("Computed field %s not found in schema", field)
			continue
		}
		if !attr.IsComputed() {
			t.Errorf("Field %s should be computed", field)
		}
	}

	// promoted_release_id is deprecated.
	if dm := s.Attributes["promoted_release_id"].GetDeprecationMessage(); dm == "" {
		t.Errorf("promoted_release_id should be marked as deprecated")
	}

	// promoted_release_ids is a list-nested block of objects with app_id and release_id.
	block, ok := s.Blocks["promoted_release_ids"]
	if !ok {
		t.Fatal("Expected promoted_release_ids to be a nested block")
	}
	nestedBlock, ok := block.(rschema.ListNestedBlock)
	if !ok {
		t.Fatal("Expected promoted_release_ids to be a ListNestedBlock")
	}
	if _, hasAppID := nestedBlock.NestedObject.Attributes["app_id"]; !hasAppID {
		t.Fatal("Expected promoted_release_ids objects to have app_id field")
	}
	if _, hasReleaseID := nestedBlock.NestedObject.Attributes["release_id"]; !hasReleaseID {
		t.Fatal("Expected promoted_release_ids objects to have release_id field")
	}

	// targets is a set attribute.
	if _, ok := s.Attributes["targets"].(rschema.SetAttribute); !ok {
		t.Errorf("targets field should be a SetAttribute")
	}
}
