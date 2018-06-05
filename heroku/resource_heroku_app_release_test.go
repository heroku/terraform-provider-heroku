package heroku

import (
	"context"
	"fmt"
	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

func TestAccHerokuAppRelease_Basic(t *testing.T) {
	var appRelease heroku.Release

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	slugId := os.Getenv("HEROKU_SLUG_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)

			if slugId == "" {
				t.Skip("HEROKU_SLUG_ID is not set; skipping test.")
			}
		},

		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppRelease_Basic(appName, slugId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppReleaseExists("heroku_app_release.foobar-release", &appRelease),
					testAccCheckHerokuAppReleaseSlugIdAttribute(&appRelease, slugId),
					resource.TestCheckResourceAttr(
						"heroku_app_release.foobar-release", "slug_id", slugId),
				),
			},
		},
	})
}

func TestAccHerokuAppRelease_OrgBasic(t *testing.T) {
	var appRelease heroku.Release

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_ORGANIZATION")
	slugId := os.Getenv("HEROKU_SLUG_ID")
	desc := fmt.Sprintf("some release description %s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
			if slugId == "" {
				t.Skip("HEROKU_SLUG_ID is not set; skipping test.")
			}
		},

		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppRelease_OrgBasic(appName, org, slugId, desc),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppReleaseExists("heroku_app_release.foobar-release", &appRelease),
					testAccCheckHerokuAppReleaseSlugIdAttribute(&appRelease, slugId),
					resource.TestCheckResourceAttr(
						"heroku_app_release.foobar-release", "slug_id", slugId),
					resource.TestCheckResourceAttr(
						"heroku_app_release.foobar-release", "description", desc),
				),
			},
		},
	})
}

func testAccCheckHerokuAppReleaseSlugIdAttribute(appRelease *heroku.Release, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if appRelease.Slug.ID != n {
			return fmt.Errorf("[ERROR] App Release slug id incorrect. Found: %s | Expected: %s", appRelease.Slug.ID, n)
		}

		return nil
	}
}

func testAccCheckHerokuAppReleaseExists(n string, appRelease *heroku.Release) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("[ERROR] Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("[ERROR] No App Release Id Set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundAppRelease, err := client.ReleaseInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundAppRelease.ID != rs.Primary.ID {
			return fmt.Errorf("[ERROR] Team Collaborator not found")
		}

		*appRelease = *foundAppRelease

		return nil
	}
}

func testAccCheckHerokuAppRelease_Basic(appName, slugId string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name = "%s"
	region = "us"
}
resource "heroku_app_release" "foobar-release" {
	app = "${heroku_app.foobar.name}"
	slug_id = "%s"
}
`, appName, slugId)
}

func testAccCheckHerokuAppRelease_OrgBasic(appName, org, slugId, desc string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name = "%s"
	region = "us"
	organization = {
		name = "%s"
	}
}
resource "heroku_app_release" "foobar-release" {
	app = "${heroku_app.foobar.name}"
	slug_id = "%s"
	description = "%s"
}
`, appName, org, slugId, desc)
}
