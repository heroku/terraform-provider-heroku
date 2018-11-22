package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/heroku-go/v3"
	"testing"
)

func TestAccHerokuAppConfigVars_basic(t *testing.T) {
	var appConfigVars heroku.ConfigVarInfoForAppResult

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfigVar_Basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppConfigVarExists("heroku_app_config_var.foobar-configs", &appConfigVars),
					resource.TestCheckResourceAttr(
						"heroku_app_config_var.foobar-configs", "public.0.ENVIRONMENT", "production"),
					resource.TestCheckResourceAttr(
						"heroku_app_config_var.foobar-configs", "private.0.PRIVATE_KEY", "some private key chain"),
					resource.TestCheckResourceAttr(
						"heroku_app_config_var.foobar-configs", "all_config_vars.%", "4"),
				),
			},
		},
	})
}

func testAccCheckHerokuAppConfigVarExists(n string, appConfigVar *heroku.ConfigVarInfoForAppResult) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Config Var ID set")
		}

		client := testAccProvider.Meta().(*Config).Api

		appName := rs.Primary.Attributes["app"]
		foundAppConfigVar, err := client.ConfigVarInfoForApp(context.TODO(), appName)
		if err != nil {
			return err
		}

		*appConfigVar = foundAppConfigVar

		return nil
	}
}

func testAccCheckHerokuAppConfigVar_Basic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_app_config_var" "foobar-configs" {
    app = "${heroku_app.foobar.name}"

    public {
		ENVIRONMENT = "production"
		USER = "foobar-user"
	}

	private {
		DATABASE_URL = "some.secret.url"
		PRIVATE_KEY = "some private key chain"
	}
}

`, appName)
}

func testAccCheckHerokuAppConfigVar_Duplicates(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_app_config_var" "foobar-configs" {
    app = "${heroku_app.foobar.name}"

    public {
		ENVIRONMENT = "production"
		USER = "foobar-user"
		PRIVATE_KEY = "some private key chain"
	}
}

`, appName)
}
