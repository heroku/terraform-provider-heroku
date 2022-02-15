package heroku

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuAppFeature(t *testing.T) {
	var feature heroku.AppFeature
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuFeatureDestroy,
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
	p := Provider()
	d := schema.TestResourceDataRaw(t, p.Schema, nil)

	client, err := providerConfigure(d)
	if err != nil {
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

	c := client.(*Config).Api
	c.URL = srv.URL

	existing := map[string]interface{}{
		"id":  "test-app:45b2b2df-2094-40d2-b099-986d0d6d8444",
		"app": "test-app",
	}
	expected := map[string]interface{}{
		"id":     expectedID + ":45b2b2df-2094-40d2-b099-986d0d6d8444",
		"app":    "test-app",
		"app_id": expectedID,
	}
	actual, err := upgradeHerokuAppFeatureV1(context.Background(), existing, client)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func testAccCheckHerokuFeatureDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

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

		client := testAccProvider.Meta().(*Config).Api

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
