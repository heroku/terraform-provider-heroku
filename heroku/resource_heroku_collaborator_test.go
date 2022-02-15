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

func TestAccHerokuCollaborator_Basic(t *testing.T) {
	var collaborator heroku.Collaborator

	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	testUser := testAccConfig.GetNonAdminUserOrAbort(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCollaborator_Basic(appName, testUser),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCollaboratorExists("heroku_collaborator.foobar-collaborator", &collaborator),
					testAccCheckHerokuCollaboratorEmailAttribute(&collaborator, testUser),
				),
			},
		},
	})
}

func testAccCheckHerokuCollaboratorExists(n string, collaborator *heroku.Collaborator) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("[ERROR] Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("[ERROR] No Collaborator Set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundCollaborator, err := client.CollaboratorInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundCollaborator.ID != rs.Primary.ID {
			return fmt.Errorf("[ERROR] Collaborator not found")
		}

		*collaborator = *foundCollaborator

		return nil
	}
}

func testAccCheckHerokuCollaboratorEmailAttribute(collaborator *heroku.Collaborator, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if collaborator.User.Email != n {
			return fmt.Errorf("[ERROR] Collaborator's email incorrect. Found: %s | Expected: %s", collaborator.User.Email, n)
		}

		return nil
	}
}

func testAccCheckHerokuCollaborator_Basic(appName, testUser string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}
resource "heroku_collaborator" "foobar-collaborator" {
	app_id = heroku_app.foobar.id
	email = "%s"
}
`, appName, testUser)
}
