package heroku

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuApp_Basic(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-16"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app, appName, "heroku-16"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.0.FOO", "bar"),
				),
			},
		},
	})
}

func TestAccHerokuApp_Disappears(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "cedar-14"

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
	appStack := "cedar-14"
	appStack2 := "heroku-16"

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
						"heroku_app.foobar", "config_vars.0.FOO", "bar"),
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
						"heroku_app.foobar", "config_vars.0.FOO", "bing"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.0.BAZ", "bar"),
				),
			},
		},
	})
}

func TestAccHerokuApp_NukeVars(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-16"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributes(&app, appName, "heroku-16"),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttr(
						"heroku_app.foobar", "config_vars.0.FOO", "bar"),
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
						"heroku_app.foobar", "config_vars.0.FOO"),
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
					testAccCheckHerokuAppBuildpacks(appName, false),
					resource.TestCheckNoResourceAttr("heroku_app.foobar", "buildpacks.0"),
				),
			},
		},
	})
}

func TestAccHerokuApp_Organization(t *testing.T) {
	var app heroku.OrganizationApp
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_ORGANIZATION")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_organization(appName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributesOrg(&app, appName, "", org),
				),
			},
		},
	})
}

func TestAccHerokuApp_Space(t *testing.T) {
	var app heroku.OrganizationApp
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_SPACES_ORGANIZATION")
	spaceName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_space(appName, spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExistsOrg("heroku_app.foobar", &app),
					testAccCheckHerokuAppAttributesOrg(&app, appName, spaceName, org),
				),
			},
		},
	})
}

// https://github.com/terraform-providers/terraform-provider-heroku/issues/2
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

func TestAccHerokuApp_AllConfigVars(t *testing.T) {
	var app heroku.App
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-16"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_config_vars_with_addons(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppExists("heroku_app.default", &app),
					resource.TestCheckResourceAttr(
						"heroku_app.default", "name", appName),
					resource.TestCheckResourceAttr(
						"heroku_app.default", "config_vars.0.A", "1"),
				),
			},
		},
	})
}

func testAccCheckHerokuAppDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

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
		client := testAccProvider.Meta().(*heroku.Service)

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
		client := testAccProvider.Meta().(*heroku.Service)

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
		client := testAccProvider.Meta().(*heroku.Service)

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
		client := testAccProvider.Meta().(*heroku.Service)

		results, err := client.BuildpackInstallationList(context.TODO(), appName, nil)
		if err != nil {
			return err
		}

		buildpacks := []string{}
		for _, installation := range results {
			buildpacks = append(buildpacks, installation.Buildpack.Name)
		}

		if multi {
			herokuMulti := "https://github.com/heroku/heroku-buildpack-multi-procfile"
			if len(buildpacks) != 2 || buildpacks[0] != herokuMulti || buildpacks[1] != "heroku/go" {
				return fmt.Errorf("Bad buildpacks: %v", buildpacks)
			}

			return nil
		}

		if len(buildpacks) != 1 || buildpacks[0] != "heroku/go" {
			return fmt.Errorf("Bad buildpacks: %v", buildpacks)
		}

		return nil
	}
}

func testAccCheckHerokuAppNoBuildpacks(appName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*heroku.Service)

		results, err := client.BuildpackInstallationList(context.TODO(), appName, nil)
		if err != nil {
			return err
		}

		buildpacks := []string{}
		for _, installation := range results {
			buildpacks = append(buildpacks, installation.Buildpack.Name)
		}

		if len(buildpacks) != 0 {
			return fmt.Errorf("Bad buildpacks: %v", buildpacks)
		}

		return nil
	}
}

func testAccCheckHerokuAppAttributesOrg(app *heroku.OrganizationApp, appName, space, org string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*heroku.Service)

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

		if app.BuildStack.Name != "heroku-16" {
			return fmt.Errorf("Bad stack: %s", app.BuildStack.Name)
		}

		if app.Name != appName {
			return fmt.Errorf("Bad name: %s", app.Name)
		}

		if app.Organization == nil || app.Organization.Name != org {
			return fmt.Errorf("Bad org: %v", app.Organization)
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

		client := testAccProvider.Meta().(*heroku.Service)

		foundApp, err := client.AppInfo(context.TODO(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundApp.Name != rs.Primary.ID {
			return fmt.Errorf("App not found")
		}

		*app = *foundApp

		return nil
	}
}

func testAccCheckHerokuAppExistsOrg(n string, app *heroku.OrganizationApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Name is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundApp, err := client.OrganizationAppInfo(context.TODO(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundApp.Name != rs.Primary.ID {
			return fmt.Errorf("App not found")
		}

		*app = *foundApp

		return nil
	}
}

func testAccInstallUnconfiguredBuildpack(t *testing.T, appName string) func() {
	return func() {
		client := testAccProvider.Meta().(*heroku.Service)

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
		client := testAccProvider.Meta().(*heroku.Service)

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

  config_vars {
    FOO = "bar"
  }
}`, appName, appStack)
}

func testAccCheckHerokuAppConfig_config_vars_with_addons(appName, appStack string) string {
	return fmt.Sprintf(`
resource "heroku_app" "default" {
  name   = "%s"
	stack = "%s"
  region = "us"

  config_vars {
    A = "1"
  }
}

resource "heroku_addon" "database" {
  app  = "${heroku_app.default.name}"
  plan = "heroku-postgresql:hobby-basic"
}

resource "heroku_addon" "redis" {
  app  = "${heroku_app.default.name}"
  plan = "heroku-redis:hobby-dev"
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

  config_vars {
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

  config_vars = []
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

  config_vars {
    FOO = "bar"
  }
}`, appName, org)
}

func testAccCheckHerokuAppConfig_space(appName, spaceName, org string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
	organization = "%s"
	region = "virginia"
}
resource "heroku_app" "foobar" {
  name   = "%s"
  space  = "${heroku_space.foobar.name}"
  region = "virginia"

  organization {
    name = "%s"
  }

  config_vars {
    FOO = "bar"
  }
}`, spaceName, org, appName, org)
}

func testAccCheckHerokuAppConfig_EmptyConfigVars(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"

  config_vars = [
  ]
}`, appName)
}
