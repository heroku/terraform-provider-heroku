package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceHerokuTelemetryDrain_Schema(t *testing.T) {
	resource := resourceHerokuTelemetryDrain()

	// Test required fields
	requiredFields := []string{"owner_id", "owner_type", "endpoint", "exporter_type", "signals"}
	for _, field := range requiredFields {
		if _, ok := resource.Schema[field]; !ok {
			t.Errorf("Required field %s not found in schema", field)
		}
		if !resource.Schema[field].Required {
			t.Errorf("Field %s should be required", field)
		}
	}

	// Test ForceNew fields
	forceNewFields := []string{"owner_id", "owner_type"}
	for _, field := range forceNewFields {
		if !resource.Schema[field].ForceNew {
			t.Errorf("Field %s should be ForceNew", field)
		}
	}

	// Test computed fields
	computedFields := []string{"created_at", "updated_at"}
	for _, field := range computedFields {
		if _, ok := resource.Schema[field]; !ok {
			t.Errorf("Computed field %s not found in schema", field)
		}
		if !resource.Schema[field].Computed {
			t.Errorf("Field %s should be computed", field)
		}
	}

	// Test signals field is a Set
	if resource.Schema["signals"].Type != schema.TypeSet {
		t.Errorf("signals field should be TypeSet")
	}

	// Test headers field is a Map
	if resource.Schema["headers"].Type != schema.TypeMap {
		t.Errorf("headers field should be TypeMap")
	}

	// Test headers field is optional
	if resource.Schema["headers"].Required {
		t.Errorf("headers field should be optional")
	}
}

func TestHerokuTelemetryDrainFeatureMatrix(t *testing.T) {
	// Test feature matrix for telemetry drains
	if !IsFeatureSupported("fir", "app", "otel") {
		t.Error("Fir apps should support otel")
	}

	if !IsFeatureSupported("fir", "space", "otel") {
		t.Error("Fir spaces should support otel")
	}

	if IsFeatureSupported("cedar", "app", "otel") {
		t.Error("Cedar apps should not support otel")
	}

	if IsFeatureSupported("cedar", "space", "otel") {
		t.Error("Cedar spaces should not support otel")
	}
}

// Acceptance test step for Fir telemetry drains
func testStep_AccHerokuTelemetryDrain_Generation_Fir(t *testing.T, spaceConfig, spaceName string) resource.TestStep {
	randString := acctest.RandString(5)
	appName := fmt.Sprintf("tftest-tel-drain-%s", randString)

	config := fmt.Sprintf(`%s

resource "heroku_app" "telemetry_drain_test" {
  name   = "%s"
  region = "virginia"
  space  = heroku_space.foobar.name
  
  organization {
    name = "%s"
  }
}

resource "heroku_telemetry_drain" "app_test" {
  owner_id      = heroku_app.telemetry_drain_test.id
  owner_type    = "app"
  endpoint      = "https://api.honeycomb.io/v1/traces"
  exporter_type = "otlphttp"
  signals       = ["traces", "metrics"]
  
  headers = {
    "x-honeycomb-team" = "test-key"
  }
}

resource "heroku_telemetry_drain" "space_test" {
  owner_id      = heroku_space.foobar.id
  owner_type    = "space"
  endpoint      = "https://logs.datadog.com/api/v2/logs"
  exporter_type = "otlphttp"
  signals       = ["logs"]
  
  headers = {
    "DD-API-KEY" = "test-space-key"
  }
}`,
		spaceConfig, appName, testAccConfig.GetOrganizationOrSkip(t))

	return resource.TestStep{
		Config: config,
		Check: resource.ComposeTestCheckFunc(
			// Check app-scoped telemetry drain
			resource.TestCheckResourceAttr("heroku_telemetry_drain.app_test", "owner_type", "app"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.app_test", "endpoint", "https://api.honeycomb.io/v1/traces"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.app_test", "exporter_type", "otlphttp"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.app_test", "signals.#", "2"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.app_test", "headers.x-honeycomb-team", "test-key"),
			resource.TestCheckResourceAttrSet("heroku_telemetry_drain.app_test", "id"),
			resource.TestCheckResourceAttrSet("heroku_telemetry_drain.app_test", "created_at"),
			resource.TestCheckResourceAttrSet("heroku_telemetry_drain.app_test", "updated_at"),

			// Check space-scoped telemetry drain
			resource.TestCheckResourceAttr("heroku_telemetry_drain.space_test", "owner_type", "space"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.space_test", "endpoint", "https://logs.datadog.com/api/v2/logs"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.space_test", "exporter_type", "otlphttp"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.space_test", "signals.#", "1"),
			resource.TestCheckResourceAttr("heroku_telemetry_drain.space_test", "headers.DD-API-KEY", "test-space-key"),
			resource.TestCheckResourceAttrSet("heroku_telemetry_drain.space_test", "id"),
			resource.TestCheckResourceAttrSet("heroku_telemetry_drain.space_test", "created_at"),
			resource.TestCheckResourceAttrSet("heroku_telemetry_drain.app_test", "updated_at"),
		),
	}
}
