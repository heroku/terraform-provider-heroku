package heroku

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	heroku "github.com/heroku/heroku-go/v6"
)

func TestAccHerokuAppFeature(t *testing.T) {
	var feature heroku.AppFeature
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHerokuFeatureDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFeature_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuFeatureExists("heroku_app_feature.runtime_metrics", &feature),
					testAccCheckHerokuFeatureEnabled(&feature, true),
					resource.TestCheckResourceAttr(
						"heroku_app_feature.runtime_metrics", "enabled", "true",
					),
				),
			},
			{
				Config: testAccCheckHerokuFeature_disabled(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuFeatureExists("heroku_app_feature.runtime_metrics", &feature),
					testAccCheckHerokuFeatureEnabled(&feature, false),
					resource.TestCheckResourceAttr(
						"heroku_app_feature.runtime_metrics", "enabled", "false",
					),
				),
			},
		},
	})
}

func TestResourceHerokuAppFeatureStateUpgradeV0(t *testing.T) {
	config := NewConfig()
	if err := config.initializeAPI(); err != nil {
		t.Fatal(err)
	}

	expectedID := "5278d60a-bb29-4f72-8936-41991e01d71e"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, writeErr := w.Write([]byte(`{"id":"` + expectedID + `"}`))
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}))
	defer srv.Close()

	config.Api.URL = srv.URL

	// The framework state upgrader resolves the fuzzy "app" (name or UUID) from
	// v0 state into the "app_id" UUID via resolveAppToAppID. Verify a name
	// resolves to its UUID through the API (the substantive v0->v1 behavior).
	appID, err := resolveAppToAppID(context.Background(), config, "test-app", "")
	if err != nil {
		t.Fatalf("error upgrading state: %s", err)
	}

	if appID != expectedID {
		t.Fatalf("expected app_id %q, got %q", expectedID, appID)
	}
}

func testAccCheckHerokuFeatureDestroy(s *terraform.State) error {
	client := testAccProviderConfig.Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_app_feature" {
			continue
		}

		_, err := client.AppFeatureInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Feature still exists")
		}
	}

	return nil
}

func testAccCheckHerokuFeatureExists(n string, feature *heroku.AppFeature) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No feature ID is set")
		}

		app, id, _ := parseCompositeID(rs.Primary.ID)
		if app != rs.Primary.Attributes["app_id"] {
			return fmt.Errorf("Bad app: %s", app)
		}

		client := testAccProviderConfig.Api

		foundFeature, err := client.AppFeatureInfo(context.TODO(), app, id)
		if err != nil {
			return err
		}

		if foundFeature.ID != id {
			return fmt.Errorf("Feature not found")
		}

		*feature = *foundFeature
		return nil
	}
}

func testAccCheckHerokuFeatureEnabled(feature *heroku.AppFeature, enabled bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if feature.Enabled != enabled {
			return fmt.Errorf("Bad enabled: %v", feature.Enabled)
		}

		return nil
	}
}

func testAccCheckHerokuFeature_basic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "example" {
	name = "%s"
	region = "us"
}

resource "heroku_app_feature" "runtime_metrics" {
	app_id = heroku_app.example.id
	name = "log-runtime-metrics"
}
`, appName)
}

func testAccCheckHerokuFeature_disabled(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "example" {
	name = "%s"
	region = "us"
}

resource "heroku_app_feature" "runtime_metrics" {
	app_id = heroku_app.example.id
	name = "log-runtime-metrics"
	enabled = false
}
`, appName)
}
