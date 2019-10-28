package heroku

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	heroku "github.com/heroku/heroku-go/v5"
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
						"heroku_addon.foobar", "config.url", "http://google.com"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "attachment_name", "foobar_an"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "deployhooks:http"),
				),
			},
		},
	})
}

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

func TestAccHerokuAddon_CustomName(t *testing.T) {
	var addon heroku.AddOn
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	customName := fmt.Sprintf("custom-addonname-%s", acctest.RandString(15))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_CustomName(appName, customName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "memcachier:dev"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "memcachier"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "name", customName),
				),
			},
		},
	})
}

func TestAccHerokuAddon_CustomName_Invalid(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	customName := "da.%dsadsa$d"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuAddonConfig_CustomName(appName, customName),
				ExpectError: regexp.MustCompile(`config is invalid: invalid value for name`),
			},
		},
	})
}

func TestAccHerokuAddon_CustomName_EmptyString(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	customName := ""

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuAddonConfig_CustomName(appName, customName),
				ExpectError: regexp.MustCompile(`config is invalid: 2 problems:.*`),
			},
		},
	})
}

func TestAccHerokuAddon_CustomName_FirstCharNum(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	customName := "1dasdad"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuAddonConfig_CustomName(appName, customName),
				ExpectError: regexp.MustCompile(`config is invalid: invalid value for name`),
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
	client := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_addon" {
			continue
		}

		_, err := client.Api.AddOnInfoByApp(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

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

func testAccCheckHerokuAddonExists(n string, addon *heroku.AddOn) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Addon ID is set")
		}

		client := testAccProvider.Meta().(*Config)

		foundAddon, err := client.Api.AddOnInfoByApp(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

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
		client := testAccProvider.Meta().(*Config)

		_, err := client.Api.AddOnDelete(context.TODO(), appName, addonName)
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
    attachment_name = "foobar_an"
    config = {
        url = "http://google.com"
	}
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

func testAccCheckHerokuAddonConfig_CustomName(appName, customAddonName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "memcachier"
    name = "%s"
}`, appName, customAddonName)
}
