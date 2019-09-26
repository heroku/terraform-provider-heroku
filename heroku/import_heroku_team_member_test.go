package heroku

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuTeamMember_importBasic(t *testing.T) {
	// Use Enterprise (Private Spaces) Team so that
	// "requires you to invite members" errors are avoided.
	team := testAccConfig.GetSpaceOrganizationOrSkip(t)
	testUser := testAccConfig.GetUserOrAbort(t)
	role := "member"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamMember_Org(team, testUser, role),
			},
			{
				ResourceName:      "heroku_team_member.foobar-member",
				ImportStateId:     buildCompositeID(team, testUser),
				ImportState:       true,
				ImportStateVerify: true,
				//				ImportStateVerifyIgnore: []string{"suppress_invites"},
			},
		},
	})
}
