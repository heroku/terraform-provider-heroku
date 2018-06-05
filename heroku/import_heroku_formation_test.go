package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"os"
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
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugId),
			},
			{
				ResourceName:      "heroku_formation.foobar-web",
				ImportStateId:     appName + ":" + formationType,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
