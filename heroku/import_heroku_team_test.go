package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccHerokuTeamImport(t *testing.T) {
	teamName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,

		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuTeam_Basic(teamName),
			},
			{
				ResourceName:      "heroku_team.foobar",
				ImportStateId:     teamName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
