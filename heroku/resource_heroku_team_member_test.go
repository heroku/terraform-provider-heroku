package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuTeamMember_Org(t *testing.T) {
	team := testAccConfig.GetAnyOrganizationOrSkip(t)
	testUser := testAccConfig.GetUserOrSkip(t)
	role := "member"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamMember_Org(team, testUser, role),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuTeamMemberExists("heroku_team_member.foobar-member"),
					resource.TestCheckResourceAttr(
						"heroku_team_member.foobar-member", "role", "member"),
				),
			},
		},
	})
}

func testAccCheckHerokuTeamMemberExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("[ERROR] Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("[ERROR] No Team Member Set")
		}

		team, email, _ := parseCompositeID(rs.Primary.ID)
		client := testAccProvider.Meta().(*Config).Api

		members, err := client.TeamMemberList(context.TODO(), team, &heroku.ListRange{Field: "email"})
		if err != nil {
			return err
		}

		var foundMember heroku.TeamMember
		for _, member := range members {
			if member.Email == email {
				foundMember = member
				break
			}
		}

		if foundMember.ID == "" {
			return fmt.Errorf("Could not find member record for %s on team %s", email, team)
		}

		return nil
	}
}

func testAccCheckHerokuTeamMember_Org(team, testUser, role string) string {
	return fmt.Sprintf(`
resource "heroku_team_member" "foobar-member" {
	team  = "%s"
	email = "%s"
	role  = "%s"
}
`, team, testUser, role)
}
