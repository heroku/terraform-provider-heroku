package heroku

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDatasourceHerokuApp_Basic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-16"
	gitUrl := fmt.Sprintf("https://git.heroku.com/%s.git", appName)
	webUrl := fmt.Sprintf("https://%s.herokuapp.com/", appName)
	herokuHostname := fmt.Sprintf("%s.herokuapp.com", appName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppWithDatasource_basic(appName, appStack),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttrSet(
						"data.heroku_app.foobar", "id"),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "stack", appStack),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "region", "us"),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "git_url", gitUrl),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "web_url", webUrl),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "config_vars.FOO", "bar"),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "buildpacks.0", "https://github.com/heroku/heroku-buildpack-multi-procfile"),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "acm", "false"),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "heroku_hostname", herokuHostname),
				),
			},
		},
	})
}

func TestAccDatasourceHerokuApp_Advanced(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	spaceName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_SPACES_ORGANIZATION")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_SPACES_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuApp_advanced(appName, spaceName, org),
			},
			{
				Config: testAccCheckHerokuAppWithDatasource_advanced(appName, spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "name", appName),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "organization.0.name", org),
					resource.TestCheckResourceAttr(
						"data.heroku_app.foobar", "space", spaceName),
				),
			},
		},
	})
}

func testAccCheckHerokuApp_basic(appName string, stack string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  stack = "%s"
  region = "us"

  buildpacks = [
    "heroku/go"
  ]

	config_vars = {
    FOO = "bar"
	}
}
`, appName, stack)
}

func testAccCheckHerokuAppWithDatasource_basic(appName string, stack string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  stack = "%s"
  region = "us"

  buildpacks = [
    "https://github.com/heroku/heroku-buildpack-multi-procfile",
    "heroku/go"
	]
	
	config_vars = {
    FOO = "bar"
	}
}

data "heroku_app" "foobar" {
  name = "${heroku_app.foobar.name}"
}
`, appName, stack)
}

func testAccCheckHerokuApp_advanced(appName, spaceName, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
	region = "virginia"
	trusted_ip_ranges = [ "0.0.0.0/0" ]
}

resource "heroku_app" "foobar" {
  name   = "%s"
  space  = "${heroku_space.foobar.name}"
  organization {
    name = "%s"
  }
  region = "virginia"
}
`, spaceName, orgName, appName, orgName)
}

func testAccCheckHerokuAppWithDatasource_advanced(appName, spaceName, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
	region = "virginia"
	trusted_ip_ranges = [ "0.0.0.0/0" ]
}

resource "heroku_app" "foobar" {
  name   = "%s"
  space  = "${heroku_space.foobar.name}"
  organization {
    name = "%s"
  }
  region = "virginia"
}

data "heroku_app" "foobar" {
  name = "${heroku_app.foobar.name}"
}
`, spaceName, orgName, appName, orgName)
}
