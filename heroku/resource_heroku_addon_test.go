package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuAddon_Basic(t *testing.T) {
	var addon heroku.AddOn
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "deployhooks:http"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "config.0.url", "http://google.com"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "deployhooks:http"),
				),
			},
		},
	})
}

// GH-198
func TestAccHerokuAddon_noPlan(t *testing.T) {
	var addon heroku.AddOn
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_no_plan(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "memcachier:dev"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "memcachier"),
				),
			},
			{
				Config: testAccCheckHerokuAddonConfig_no_plan(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "memcachier:dev"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "memcachier"),
				),
			},
		},
	})
}

/**
* When an addon is attached to an app, its useful to see the values of config vars attached to said heroku_app resource without
* having to do a state refresh on the heroku_app resource. Instead, embed that info into the heroku_addon.
**/
func TestAccHerokuAddon_ConfigAdded(t *testing.T) {
	var addon heroku.AddOn
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_hostedgraphite(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuConfigVarExists("heroku_addon.foobar", appName, addon.ID, "HOSTEDGRAPHITE_APIKEY"),
				),
			},
		},
	})
}

func TestAccHerokuAddon_Disappears(t *testing.T) {
	var addon heroku.AddOn
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonDisappears(appName, "deployhooks"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckHerokuAddonDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_addon" {
			continue
		}

		_, err := client.AddOnInfoByApp(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Addon still exists")
		}
	}

	return nil
}

func testAccCheckHerokuAddonAttributes(addon *heroku.AddOn, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if addon.Plan.Name != n {
			return fmt.Errorf("Bad plan: %s", addon.Plan.Name)
		}

		return nil
	}
}

func testAccCheckHerokuConfigVarExists(resourceName string, appName string, addonId string, configVar string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rootModule := s.RootModule()
		resource, ok := rootModule.Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if resource.Primary.ID == "" {
			return fmt.Errorf("No addon Id found %s", resourceName)
		}

		client := testAccProvider.Meta().(*heroku.Service)

		configVarInfo, err := client.ConfigVarInfoForApp(context.TODO(), appName)

		if err != nil {
			return err
		}
		if configVarInfo == nil {
			return nil
		}
		configValue := resource.Primary.Attributes["all_config_vars."+configVar]
		if configValue != *configVarInfo[configVar] {
			return fmt.Errorf("Mismatched config value set %s, actual: %s expected: %s", configVar, configValue, *configVarInfo[configVar])
		} else {
			//happy path, expected config key found, and value matches
			return nil
		}
	}
}

func testAccCheckHerokuAddonExists(n string, addon *heroku.AddOn) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Addon ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundAddon, err := client.AddOnInfoByApp(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundAddon.ID != rs.Primary.ID {
			return fmt.Errorf("Addon not found")
		}

		*addon = *foundAddon

		return nil
	}
}

func testAccCheckHerokuAddonDisappears(appName, addonName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*heroku.Service)

		_, err := client.AddOnDelete(context.TODO(), appName, addonName)
		return err
	}
}

func testAccCheckHerokuAddonConfig_basic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "deployhooks:http"
    config {
        url = "http://google.com"
    }
}`, appName)
}

func testAccCheckHerokuAddonConfig_hostedgraphite(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "hostedgraphite:free"
}`, appName)
}

func testAccCheckHerokuAddonConfig_no_plan(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "memcachier"
}`, appName)
}
