package heroku

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatasourceHerokuTeamMembers_Basic(t *testing.T) {
	teamName := testAccConfig.GetTeamOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

// TestAccDatasourceHerokuTeamMembers_InvalidRole verifies the restored SDKv2
// StringInSlice(TeamMemberRoles) validation rejects an unknown role at plan time.
func TestAccDatasourceHerokuTeamMembers_InvalidRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "heroku_team_members" "foobar" {
  team  = "tftest"
  roles = ["member", "bogus"]
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

// TestAccDatasourceHerokuTeamMembers_EmptyRoles verifies the restored SDKv2
// MinItems: 1 constraint rejects an empty roles filter at plan time.
func TestAccDatasourceHerokuTeamMembers_EmptyRoles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "heroku_team_members" "foobar" {
  team  = "tftest"
  roles = []
}
`,
				ExpectError: regexp.MustCompile(`at least 1`),
			},
		},
	})
}
