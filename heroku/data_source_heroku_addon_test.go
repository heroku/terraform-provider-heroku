package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDatasourceHerokuAddon_Basic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonBasic(appName),
			},
			{
				Config: testAccCheckHerokuAddonWithDatasourceBasic(appName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.heroku_addon.test_data", "app", appName),
					resource.TestCheckResourceAttr(
						"data.heroku_addon.test_data", "plan", "deployhooks:http"),
				),
			},
		},
	})
}

func testAccCheckHerokuAddonBasic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "deployhooks:http"
    config = {
		url = "http://google.com"
	}
}
`, appName)
}

func testAccCheckHerokuAddonWithDatasourceBasic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "deployhooks:http"
    config = {
		url = "http://google.com"
	}
}

data "heroku_addon" "test_data" {
  name = "${heroku_addon.foobar.id}"
}
`, appName)
}
