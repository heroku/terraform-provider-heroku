package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccHerokuConfigAssociation_Basic(t *testing.T) {
	org := testAccConfig.GetOrganizationOrSkip(t)
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfigAssociation_Basic(org, appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuConfigAssociationExists("heroku_config_association.foobar-config", "RAILS_ENV", "PRIVATE_KEY"),
					resource.TestCheckResourceAttr(
						"heroku_config_association.foobar-config", "vars.RAILS_ENV", "PROD"),
					resource.TestCheckResourceAttr(
						"heroku_config_association.foobar-config", "sensitive_vars.PRIVATE_KEY", "it_is_a_secret"),
				),
			},
		},
	})
}

func TestAccHerokuConfigAssociation_Advanced(t *testing.T) {
	org := testAccConfig.GetOrganizationOrSkip(t)
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	configName := fmt.Sprintf("config-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfigAssociation_Advanced(org, appName, configName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuConfigAssociationExists("heroku_config_association.foobar-config", "RAILS_ENV", "PRIVATE_KEY"),
					resource.TestCheckResourceAttr(
						"heroku_config_association.foobar-config", "vars.RAILS_ENV", "PROD"),
				),
			},
		},
	})
}

func testAccCheckHerokuConfigAssociationExists(n string, vars ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("config association not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no config association ID set")
		}

		client := testAccProvider.Meta().(*Config).Api

		app := rs.Primary.Attributes["app_id"]
		remoteConfig, err := client.ConfigVarInfoForApp(context.TODO(), app)
		if err != nil {
			return err
		}

		for _, variable := range vars {
			if _, ok := remoteConfig[variable]; !ok {
				return fmt.Errorf("Config var %s doesn't exist on app %s", variable, app)
			}
		}

		return nil
	}
}

func testAccCheckHerokuConfigAssociation_Basic(org, appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
  organization {
    name = "%s"
  }
}

resource "heroku_config_association" "foobar-config" {
    app_id = "${heroku_app.foobar.name}"

    vars = {
       RAILS_ENV = "PROD"
       LOG_LEVEL = "DEBUG"
    }

    sensitive_vars = {
        PRIVATE_KEY = "it_is_a_secret"
        API_TOKEN   = "some_token"
    }
}`, appName, org)
}

func testAccCheckHerokuConfigAssociation_Advanced(org, appName, configName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
  organization {
    name = "%s"
  }
}

resource "heroku_config" "config" {
    name = "%s"

    vars = {
       RAILS_ENV = "PROD"
       LOG_LEVEL = "DEBUG"
    }

    sensitive_vars = {
        PRIVATE_KEY = "it_is_a_secret"
        API_TOKEN   = "some_token"
    }
}

resource "heroku_config_association" "foobar-config" {
    app_id = "${heroku_app.foobar.name}"

    vars = "${heroku_config.config.vars}"
    sensitive_vars = "${heroku_config.config.sensitive_vars}"
}`, appName, org, configName)
}
