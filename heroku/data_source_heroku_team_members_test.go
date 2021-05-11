package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDatasourceHerokuTeamMembers_Basic(t *testing.T) {
	teamName := testAccConfig.GetTeamOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamMembersWithDataSource_Basic(teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.heroku_team_members.foobar", "team", teamName),
					resource.TestCheckResourceAttr("data.heroku_team_members.foobar", "roles.#", "4"),
					resource.TestCheckResourceAttrSet("data.heroku_team_members.foobar", "members.#"),
				),
			},
		},
	})
}

func testAccCheckHerokuTeamMembersWithDataSource_Basic(teamName string) string {
	return fmt.Sprintf(`
data "heroku_team_members" "foobar" {
  team = "%s"
  roles = ["admin", "member", "viewer", "collaborator"]
}
`, teamName)
}
