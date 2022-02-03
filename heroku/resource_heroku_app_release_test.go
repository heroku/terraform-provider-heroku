package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuAppRelease_Basic(t *testing.T) {
	var appRelease heroku.Release

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	slugID := testAccConfig.GetSlugIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppRelease_Basic(appName, slugID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppReleaseExists("heroku_app_release.foobar-release", &appRelease),
					resource.TestCheckResourceAttr(
						"heroku_app_release.foobar-release", "slug_id", slugID),
				),
			},
		},
	})
}

func TestAccHerokuAppRelease_OrgBasic(t *testing.T) {
	var appRelease heroku.Release

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)
	slugID := testAccConfig.GetSlugIDOrSkip(t)
	desc := fmt.Sprintf("some release description %s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppRelease_OrgBasic(appName, org, slugID, desc),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppReleaseExists("heroku_app_release.foobar-release", &appRelease),
					resource.TestCheckResourceAttr(
						"heroku_app_release.foobar-release", "slug_id", slugID),
					resource.TestCheckResourceAttr(
						"heroku_app_release.foobar-release", "description", desc),
				),
			},
		},
	})
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

		client := testAccProvider.Meta().(*Config).Api

		foundAppRelease, err := client.ReleaseInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundAppRelease.ID != rs.Primary.ID {
			return fmt.Errorf("[ERROR] App release not found")
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
	app_id = heroku_app.foobar.id
	slug_id = "%s"
}
`, appName, slugId)
}

func testAccCheckHerokuAppRelease_OrgBasic(appName, org, slugId, desc string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name = "%s"
	region = "us"
	organization {
		name = "%s"
	}
}
resource "heroku_app_release" "foobar-release" {
	app_id = heroku_app.foobar.id
	slug_id = "%s"
	description = "%s"
}
`, appName, org, slugId, desc)
}
