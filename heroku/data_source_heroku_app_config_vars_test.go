package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	heroku "github.com/heroku/heroku-go/v3"
)

func TestAccDatasourceHerokuAppConfigVars(t *testing.T) {
	appName := fmt.Sprintf("tftest-app-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfigVar(appName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_app.app", "name", appName),
					testAccCheckHerokuConfigVarExists(
						"data.heroku_app_config_vars.app_config_vars", appName, "HOSTEDGRAPHITE_APIKEY"),
					testAccCheckHerokuConfigVarExists(
						"heroku_app.app2", appName, "HOSTEDGRAPHITE_APIKEY"),
				),
			},
		},
	})
}

func testAccCheckHerokuAppConfigVar(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "app" {
  name         = "%s"
  region       = "us"
  stack        = "heroku-16"
}

resource "heroku_addon" "hostedgraphite" {
	app = "${heroku_app.app.name}"
	plan = "hostedgraphite:free"
}

data "heroku_app_config_vars" "app_config_vars" {
  app = "${heroku_app.app.name}"
  depends = ["${heroku_addon.hostedgraphite.app}"]
}

resource "heroku_app" "app2" {
	name         = "%s-2"
	region       = "us"
	stack        = "heroku-16"
	config_vars {
		HOSTEDGRAPHITE_APIKEY = "${data.heroku_app_config_vars.app_config_vars.all_config_vars["HOSTEDGRAPHITE_APIKEY"]}"
	}
}
`, appName, appName)
}

func testAccCheckHerokuConfigVarExists(resourceName string, appName string, configVar string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rootModule := s.RootModule()
		resource, ok := rootModule.Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if resource.Primary.ID == "" {
			return fmt.Errorf("No resource id found %s", resourceName)
		}
		client := testAccProvider.Meta().(*heroku.Service)
		configVarInfo, err := client.ConfigVarInfoForApp(context.TODO(), appName)
		if err != nil {
			return err
		}
		if configVarInfo == nil {
			return fmt.Errorf("No config vars found for %s", resourceName)
		}
		configValue := resource.Primary.Attributes["all_config_vars."+configVar]
		if configValue != *configVarInfo[configVar] {
			return fmt.Errorf("Mismatched config value set %s, actual: %s expected: %s", configVar, configValue, *configVarInfo[configVar])
		}
		//happy path, expected config key found, and value matches
		return nil
	}
}
