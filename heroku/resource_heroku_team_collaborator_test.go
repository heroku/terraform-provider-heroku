package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

func TestAccHerokuTeamCollaborator_Org(t *testing.T) {
	var teamCollaborator heroku.TeamAppCollaborator

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_ORGANIZATION")
	testUser := os.Getenv("HEROKU_TEST_USER")
	perms := "[\"deploy\", \"operate\", \"view\"]"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}

			if testUser == "" {
				t.Skip("HEROKU_TEST_USER is not set; skipping test.")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamCollaborator_Org(org, appName, testUser, perms),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuTeamCollaboratorExists("heroku_team_collaborator.foobar-collaborator", &teamCollaborator),
					testAccCheckHerokuTeamCollaboratorEmailAttribute(&teamCollaborator, testUser),
					resource.TestCheckResourceAttr(
						"heroku_team_collaborator.foobar-collaborator", "permissions.1056122515", "deploy"),
				),
			},
		},
	})
}

func TestAccHerokuTeamCollaboratorPermsOutOfOrder_Org(t *testing.T) {
	var teamCollaborator heroku.TeamAppCollaborator

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_ORGANIZATION")
	testUser := os.Getenv("HEROKU_TEST_USER")
	perms := "[\"view\", \"operate\", \"deploy\"]"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}

			if testUser == "" {
				t.Skip("HEROKU_TEST_USER is not set; skipping test.")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamCollaborator_Org(org, appName, testUser, perms),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuTeamCollaboratorExists("heroku_team_collaborator.foobar-collaborator", &teamCollaborator),
					testAccCheckHerokuTeamCollaboratorEmailAttribute(&teamCollaborator, testUser),
					resource.TestCheckResourceAttr(
						"heroku_team_collaborator.foobar-collaborator", "permissions.1056122515", "deploy"),
				),
			},
		},
	})
}

func testAccCheckHerokuTeamCollaboratorExists(n string, teamCollaborator *heroku.TeamAppCollaborator) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("[ERROR] Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("[ERROR] No Team Collaborator Set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundTeamCollaborator, err := client.TeamAppCollaboratorInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundTeamCollaborator.ID != rs.Primary.ID {
			return fmt.Errorf("[ERROR] Team Collaborator not found")
		}

		*teamCollaborator = *foundTeamCollaborator

		return nil
	}
}

func testAccCheckHerokuTeamCollaboratorEmailAttribute(teamCollaborator *heroku.TeamAppCollaborator, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if teamCollaborator.User.Email != n {
			return fmt.Errorf("[ERROR] Team Collaborator's email incorrect. Found: %s | Expected: %s", teamCollaborator.User.Email, n)
		}

		return nil
	}
}

func testAccCheckHerokuTeamCollaborator_Org(org, appName, testUser, perms string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
    organization {
        name = "%s"
    }
}
resource "heroku_team_collaborator" "foobar-collaborator" {
	app = "${heroku_app.foobar.name}"
	email = "%s"
	permissions = %s
}
`, appName, org, testUser, perms)
}
