package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestAccHerokuTeam_Basic(t *testing.T) {
	teamName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeam_Basic(teamName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuTeamExists("heroku_team.foobar"),
					resource.TestCheckResourceAttr(
						"heroku_team.foobar", "name", teamName),
					resource.TestCheckResourceAttr(
						"heroku_team.foobar", "default", "false"),
				),
			},
		},
	})
}

func testAccCheckHerokuTeamExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("[ERROR] Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("[ERROR] No Team Set")
		}

		client := testAccProvider.Meta().(*Config).Api

		team, err := client.TeamInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if team.ID != rs.Primary.ID {
			return fmt.Errorf("[ERROR] Team not found")
		}

		return nil
	}
}

func testAccCheckHerokuTeam_Basic(teamName string) string {
	return fmt.Sprintf(`
resource "heroku_team" "foobar" {
    name = "%s"
}`, teamName)
}
