package heroku

import (
	"context"
	"fmt"
	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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

		client := testAccProvider.Meta().(*heroku.Service)

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
    public = {
		ENVIRONMENT = "production"
		USER = "foobar-user"
	}

	private = {
		DATABASE_URL = "some.secret.url"
		PRIVATE_KEY = "some private key chain"
	}
}

`, appName)
}
