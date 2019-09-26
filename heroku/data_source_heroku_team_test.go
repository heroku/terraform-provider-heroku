package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccDatasourceHerokuTeam_Basic(t *testing.T) {
	// Since there will not be a heroku_team resource, this test will require an existing team for execution.
	teamName := testAccConfig.GetTeamOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamWithDataSource_Basic(teamName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.heroku_team.foobar", "name", teamName),
				),
			},
		},
	})
}

func testAccCheckHerokuTeamWithDataSource_Basic(teamName string) string {
	return fmt.Sprintf(`

data "heroku_team" "foobar" {
  name = "%s"
}
`, teamName)
}
