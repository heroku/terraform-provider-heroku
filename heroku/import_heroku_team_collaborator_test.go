package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"os"
)

func TestAccHerokuTeamCollaborator_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := os.Getenv("HEROKU_ORGANIZATION")
	testUser := os.Getenv("HEROKU_TEST_USER")
	perms := "[\"deploy\", \"operate\", \"view\"]"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeamCollaborator_Org(org, appName, testUser, perms),
			},
			{
				ResourceName:            "heroku_team_collaborator.foobar-collaborator",
				ImportStateId:           appName + ":" + testUser,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"suppress_invites"},
			},
		},
	})
}
