package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/fsouza/go-dockerclient/external/github.com/Sirupsen/logrus"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

func TestAccHerokuFormationSingleUpdate_WithOrg(t *testing.T) {
	var formation heroku.Formation

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	slugId := os.Getenv("HEROKU_SLUG_ID")
	org := os.Getenv("HEROKU_ORGANIZATION")

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
				Config: testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugId),
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

func testAccCheckHerokuFormationExists(n string, formation *heroku.Formation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		logrus.Printf("What is ok: %s", ok)

		if !ok {
			return fmt.Errorf("Formation not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Formation ID set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundFormation, err := client.FormationInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

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

func testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugId string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
  organization {
    name = "%s"
  }
}
resource "heroku_app_release" "foobar-release" {
	app = "${heroku_app.foobar.name}"
	slug_id = "%s"
}
resource "heroku_formation" "foobar-web" {
	app = "${heroku_app.foobar.name}"
	type = "web"
	quantity = 2
	size = "standard-2x"
}
`, appName, org, slugId)
}
