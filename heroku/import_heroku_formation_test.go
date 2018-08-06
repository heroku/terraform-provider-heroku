package heroku

import (
	"fmt"
	"testing"

	"os"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuFormation_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	slugId := os.Getenv("HEROKU_SLUG_ID")
	org := os.Getenv("HEROKU_ORGANIZATION")
	formationType := "web"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if slugId == "" {
				t.Skip("HEROKU_SLUG_ID is not set; skipping test.")
			}
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugId, "standard-2x", 2),
			},
			{
				ResourceName:      "heroku_formation.foobar-web",
				ImportStateId:     buildCompositeID(appName, formationType),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
