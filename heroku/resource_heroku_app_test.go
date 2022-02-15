package heroku

import (
	"context"
	"fmt"
	"log"
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

func TestAccHerokuApp_Basic(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-20"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app, appName, "heroku-20"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttrSet(
						"heroku_app.foobar", "uuid"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bar"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "internal_routing", "false"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "all_config_vars.%", "1"),
				),
			},
		},
	})
}

func TestAccHerokuApp_DontSetAllConfigVars(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-20"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_DontSetConfigVars(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttrSet(
						"heroku_app.foobar", "uuid"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bar"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "internal_routing", "false"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "all_config_vars.%", "0"),
				),
			},
		},
	})
}

func TestAccHerokuApp_Disappears(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-18"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppDisappears(appName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccHerokuApp_Change(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appName2 := fmt.Sprintf("%s-v2", appName)
	appStack := "heroku-18"
	appStack2 := "heroku-20"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app, appName, appStack),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "stack", appStack),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bar"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_updated(appName2, appStack2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributesUpdated(&app, appName2, appStack2),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName2),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "stack", appStack2),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bing"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.BAZ", "bar"),
				),
			},
		},
	})
}

func TestAccHerokuApp_NukeVars(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-20"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app, appName, "heroku-20"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bar"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_no_vars(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributesNoVars(&app, appName),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckNoResourceAttr(
						"heroku_app.foobar", "config_vars.FOO"),
				),
			},
		},
	})
}

func TestAccHerokuApp_Buildpacks(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_go(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppBuildpacks(appName, false),
					resource.TestCheckResourceAttr("heroku_app.foobar", "buildpacks.0", "heroku/go"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_multi(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppBuildpacks(appName, true),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "buildpacks.0", "https://github.com/heroku/heroku-buildpack-multi-procfile"),
					resource.TestCheckResourceAttr("heroku_app.foobar", "buildpacks.1", "heroku/go"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_no_vars(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppNoBuildpacks(appName),
					resource.TestCheckNoResourceAttr("heroku_app.foobar", "buildpacks.0"),
				),
			},
		},
	})
}

func TestAccHerokuApp_ExternallySetBuildpacks(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_no_vars(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppNoBuildpacks(appName),
					resource.TestCheckNoResourceAttr("heroku_app.foobar", "buildpacks.0"),
				),
			},
			{
				PreConfig: testAccInstallUnconfiguredBuildpack(t, appName),
				Config:    testAccCheckHerokuAppConfig_no_vars(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckNoResourceAttr("heroku_app.foobar", "buildpacks.0"),
				),
			},
		},
	})
}

func TestAccHerokuApp_ACM(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_organization(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr("heroku_app.foobar", "acm", "false"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_acm_enabled(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr("heroku_app.foobar", "acm", "true"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_acm_disabled(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr("heroku_app.foobar", "acm", "false"),
				),
			},
		},
	})
}

func TestAccHerokuApp_Organization(t *testing.T) {
	var app heroku.TeamApp
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_organization(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributesOrg(&app, appName, "", org, false),
				),
			},
		},
	})
}

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccHerokuApp_Space(t *testing.T, spaceConfig, spaceName string) resource.TestStep {
	var app heroku.TeamApp
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetSpaceOrganizationOrSkip(t)

	return resource.TestStep{
		Config: testAccCheckHerokuAppConfig_space(spaceConfig, appName, org),
		Check: resource.ComposeTestCheckFunc(
			testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
			testAccCheckHerokuAppAttributesOrg(&app, appName, spaceName, org, false),
		),
	}
}

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccHerokuApp_Space_Internal(t *testing.T, spaceConfig, spaceName string) resource.TestStep {
	var app heroku.TeamApp
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetSpaceOrganizationOrSkip(t)

	return resource.TestStep{
		Config: testAccCheckHerokuAppConfig_space_internal(spaceConfig, appName, org),
		Check: resource.ComposeTestCheckFunc(
			testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
			testAccCheckHerokuAppAttributesOrg(&app, appName, spaceName, org, true),
		),
	}
}

// https://github.com/heroku/terraform-provider-heroku/issues/2
func TestAccHerokuApp_EmptyConfigVars(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_EmptyConfigVars(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributesNoVars(&app, appName),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
				),
			},
		},
	})
}

func TestAccHerokuApp_SensitiveConfigVars(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_Sensitive(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bar"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "sensitive_config_vars.PRIVATE_KEY", "it is a secret"),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_SensitiveUpdate(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.FOO", "bar1"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "sensitive_config_vars.PRIVATE_KEY", "it is a secret1"),
				),
			},

			{
				Config: testAccCheckHerokuAppConfig_SensitiveUpdate_Swap(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.WIDGETS", "fake"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "sensitive_config_vars.PRIVATE_KEY", "it is a secret1"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "sensitive_config_vars.FOO", "bar1"),
				),
			},
		},
	})
}

func TestAccHerokuApp_Organization_Locked(t *testing.T) {
	var app heroku.TeamApp
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_locked(appName, org, "false"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr("heroku_app.foobar", "organization.0.locked", "false"),
					resource.TestCheckResourceAttr("heroku_app.foobar", "organization.0.name", org),
				),
			},
			{
				Config: testAccCheckHerokuAppConfig_locked(appName, org, "true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
					resource.TestCheckResourceAttr("heroku_app.foobar", "organization.0.locked", "true"),
					resource.TestCheckResourceAttr("heroku_app.foobar", "organization.0.name", org),
				),
			},
		},
	})
}

func TestResourceHerokuAppStateUpgradeV0(t *testing.T) {
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
		"id": "test-app",
	}
	expected := map[string]interface{}{
		"id": expectedID,
	}
	actual, err := resourceHerokuAppStateUpgradeV0(context.Background(), existing, client)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
	}
}

func testAccCheckHerokuAppDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_app" {
			continue
		}

		_, err := client.AppInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("App still exists")
		}
	}

	return nil
}

func testAccCheckHerokuAppAttributes(app *heroku.App, appName, stackName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		if app.Region.Name != "us" {
			return fmt.Errorf("Bad region: %s", app.Region.Name)
		}

		if app.BuildStack.Name != stackName {
			return fmt.Errorf("Bad stack: %s", app.BuildStack.Name)
		}

		if app.Name != appName {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		vars, err := client.ConfigVarInfoForApp(context.TODO(), app.Name)
		if err != nil {
			return err
		}

		if vars["FOO"] == nil || *vars["FOO"] != "bar" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		return nil
	}
}

func testAccCheckHerokuAppAttributesUpdated(app *heroku.App, appName, stackName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		if app.BuildStack.Name != stackName {
			return fmt.Errorf("Bad stack: %s", app.BuildStack.Name)
		}

		if app.Name != appName {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		vars, err := client.ConfigVarInfoForApp(context.TODO(), app.Name)
		if err != nil {
			return err
		}

		// Make sure we kept the old one
		if vars["FOO"] == nil || *vars["FOO"] != "bing" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		if vars["BAZ"] == nil || *vars["BAZ"] != "bar" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		return nil

	}
}

func testAccCheckHerokuAppAttributesNoVars(app *heroku.App, appName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		if app.Name != appName {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		vars, err := client.ConfigVarInfoForApp(context.TODO(), app.Name)
		if err != nil {
			return err
		}

		if len(vars) != 0 {
			return fmt.Errorf("vars exist: %v", vars)
		}

		return nil
	}
}

func testAccCheckHerokuAppBuildpacks(appName string, multi bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		results, err := client.BuildpackInstallationList(context.TODO(), appName, nil)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] List of the app's buildpack installations: %v", results)

		buildpacks := make([]string, 0)
		for _, installation := range results {
			buildpacks = append(buildpacks, installation.Buildpack.Name)
		}

		log.Printf("[DEBUG] List of the buildpacks: %v", buildpacks)

		if multi {
			herokuMulti := "https://github.com/heroku/heroku-buildpack-multi-procfile"
			if len(buildpacks) != 2 || buildpacks[0] != herokuMulti || buildpacks[1] != "heroku/go" {
				return fmt.Errorf("bad buildpacks: %v", buildpacks)
			}

			return nil
		}

		if len(buildpacks) != 1 || buildpacks[0] != "heroku/go" {
			return fmt.Errorf("expected buildpack length to not equal 1 OR buildpacks[0] to not be \"heroku/go\" but got: %v", buildpacks)
		}

		return nil
	}
}

func testAccCheckHerokuAppNoBuildpacks(appName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		results, err := client.BuildpackInstallationList(context.TODO(), appName, nil)
		if err != nil {
			return err
		}

		buildpacks := make([]string, 0)
		for _, installation := range results {
			buildpacks = append(buildpacks, installation.Buildpack.Name)
		}

		if len(buildpacks) != 0 {
			return fmt.Errorf("expected 0 buildpacks but got %d: %v", len(buildpacks), buildpacks)
		}

		return nil
	}
}

func testAccCheckHerokuAppAttributesOrg(app *heroku.TeamApp, appName, space, org string, internal bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		if app.Region.Name != "us" && app.Region.Name != "virginia" {
			return fmt.Errorf("Bad region: %s", app.Region.Name)
		}

		var appSpace string
		if app.Space != nil {
			appSpace = app.Space.Name
		}

		if appSpace != space {
			return fmt.Errorf("Bad space: %s", appSpace)
		}

		// This needs to be updated whenever heroku bumps the stack number
		if app.BuildStack.Name != "heroku-20" {
			return fmt.Errorf("Bad stack: %s", app.BuildStack.Name)
		}

		if app.Name != appName {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		if app.Team == nil || app.Team.Name != org {
			return fmt.Errorf("Bad org: %v", app.Team)
		}

		appInternalRouting := false
		if app.InternalRouting != nil {
			appInternalRouting = *app.InternalRouting
		}
		if appInternalRouting != internal {
			return fmt.Errorf("Bad internal routing: %v (want %v)", appInternalRouting, internal)
		}

		vars, err := client.ConfigVarInfoForApp(context.TODO(), app.Name)
		if err != nil {
			return err
		}

		if vars["FOO"] == nil || *vars["FOO"] != "bar" {
			return fmt.Errorf("Bad config vars: %v", vars)
		}

		return nil
	}
}

func testAccCheckHerokuAppExists(n string, app *heroku.App) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundApp, err := client.AppInfo(context.TODO(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundApp.ID != rs.Primary.ID {
			return fmt.Errorf("App not found")
		}

		*app = *foundApp

		return nil
	}
}

func testAccCheckHerokuAppExistsOrg(n string, app *heroku.TeamApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundApp, err := client.TeamAppInfo(context.TODO(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundApp.ID != rs.Primary.ID {
			return fmt.Errorf("App not found")
		}

		*app = *foundApp

		return nil
	}
}

func testAccInstallUnconfiguredBuildpack(t *testing.T, appName string) func() {
	return func() {
		client := testAccProvider.Meta().(*Config).Api

		opts := heroku.BuildpackInstallationUpdateOpts{
			Updates: []struct {
				Buildpack string `json:"buildpack" url:"buildpack,key"`
			}{
				{Buildpack: "heroku/go"},
			},
		}

		_, err := client.BuildpackInstallationUpdate(context.TODO(), appName, opts)
		if err != nil {
			t.Fatalf("Error updating buildpacks: %s", err)
		}
	}
}

func testAccCheckHerokuAppDisappears(appName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Config).Api

		_, err := client.AppDelete(context.TODO(), appName)
		return err
	}
}

func testAccCheckHerokuAppConfig_basic(appName, appStack string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  stack = "%s"
  region = "us"

  config_vars = {
    FOO = "bar"
  }
}`, appName, appStack)
}

func testAccCheckHerokuAppConfig_go(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"

  buildpacks = ["heroku/go"]
}`, appName)
}

func testAccCheckHerokuAppConfig_multi(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"

  buildpacks = [
    "https://github.com/heroku/heroku-buildpack-multi-procfile",
    "heroku/go"
  ]
}`, appName)
}

func testAccCheckHerokuAppConfig_updated(appName, appStack string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
	stack  = "%s"
  region = "us"

  config_vars = {
    FOO = "bing"
    BAZ = "bar"
  }
}`, appName, appStack)
}

func testAccCheckHerokuAppConfig_no_vars(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"

  buildpacks = []

  config_vars = {}
}`, appName)
}

func testAccCheckHerokuAppConfig_organization(appName, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"

  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_space(spaceConfig, appName, org string) string {
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

resource "heroku_app" "foobar" {
  name   = "%s"
  space  = heroku_space.foobar.name
  region = "virginia"

  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar"
  }
}`, spaceConfig, appName, org)
}

func testAccCheckHerokuAppConfig_space_internal(spaceConfig, appName, org string) string {
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

resource "heroku_app" "foobar" {
  name             = "%s"
  space            = heroku_space.foobar.name
  region           = "virginia"
	internal_routing = true

  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar"
  }
}`, spaceConfig, appName, org)
}

func testAccCheckHerokuAppConfig_EmptyConfigVars(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
}`, appName)
}

func testAccCheckHerokuAppConfig_acm_enabled(appName, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
  acm = true
  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_acm_disabled(appName, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
  acm = false
  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_Sensitive(appName, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
  acm = false
  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar"
  }

  sensitive_config_vars = {
    PRIVATE_KEY = "it is a secret"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_SensitiveUpdate(appName, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
  acm = false
  organization {
    name = "%s"
  }

  config_vars = {
    FOO = "bar1"
  }

  sensitive_config_vars = {
    PRIVATE_KEY = "it is a secret1"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_SensitiveUpdate_Swap(appName, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
  acm = false
  organization {
    name = "%s"
  }

  config_vars = {
    WIDGETS = "fake"
  }

  sensitive_config_vars = {
    FOO = "bar1"
    PRIVATE_KEY = "it is a secret1"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_locked(appName, org, locked string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"

  organization {
    name = "%s"
	locked = %s
  }
}`, appName, org, locked)
}

func testAccCheckHerokuAppConfig_DontSetConfigVars(appName, appStack string) string {
	return fmt.Sprintf(`
provider "heroku" {
  customizations {
    set_app_all_config_vars_in_state = false
  }
}

resource "heroku_app" "foobar" {
  name   = "%s"
  stack = "%s"
  region = "us"

  config_vars = {
    FOO = "bar"
  }
}`, appName, appStack)
}
