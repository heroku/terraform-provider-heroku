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

func TestAccHerokuFormationSingleUpdate_WithOrg(t *testing.T) {
	var formation heroku.Formation

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)
	slugID := testAccConfig.GetSlugIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugID, "standard-2x", 2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuFormationExists("heroku_formation.foobar-web", &formation),
					testAccCheckHerokuFormationSizeAttribute(&formation, "Standard-2X"),
					resource.TestCheckResourceAttr(
						"heroku_formation.foobar-web", "size", "Standard-2X"),
					resource.TestCheckResourceAttr(
						"heroku_formation.foobar-web", "quantity", "2"),
				),
			},
		},
	})
}

func TestAccHerokuFormationUpdateFreeDyno(t *testing.T) {
	var formation heroku.Formation

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	slugID := testAccConfig.GetSlugIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFormationConfig_WithOutOrg(appName, slugID, "free", 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuFormationExists("heroku_formation.foobar-web", &formation),
					testAccCheckHerokuFormationSizeAttribute(&formation, "Free"),
					resource.TestCheckResourceAttr(
						"heroku_formation.foobar-web", "size", "Free"),
					resource.TestCheckResourceAttr(
						"heroku_formation.foobar-web", "quantity", "1"),
				),
			},
		},
	})

}

func testAccCheckHerokuFormationExists(n string, formation *heroku.Formation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Formation not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Formation ID set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundFormation, err := client.FormationInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundFormation.ID != rs.Primary.ID {
			return fmt.Errorf("Formation not found")
		}

		*formation = *foundFormation

		return nil
	}
}

func testAccCheckHerokuFormationSizeAttribute(formation *heroku.Formation, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if formation.Size != n {
			return fmt.Errorf("formation's size is not correct. Found: %s | Got: %s", formation.Size, n)
		}

		return nil
	}
}

func testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugId, dynoSize string, dynoQuant int) string {
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
}
resource "heroku_formation" "foobar-web" {
	app_id = heroku_app.foobar.id
	type = "web"
	size = "%s"
	quantity = %d
}
`, appName, org, slugId, dynoSize, dynoQuant)
}

func testAccCheckHerokuFormationConfig_WithOutOrg(appName, slugId, dynoSize string, dynoQuant int) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}
resource "heroku_app_release" "foobar-release" {
	app_id = heroku_app.foobar.id
	slug_id = "%s"
}
resource "heroku_formation" "foobar-web" {
	app_id = heroku_app.foobar.id
	type = "web"
	size = "%s"
	quantity = %d
}
`, appName, slugId, dynoSize, dynoQuant)
}
