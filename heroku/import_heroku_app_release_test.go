package heroku

import (
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"os"
)

func TestAccHerokuAppRelease_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	slugId := os.Getenv("HEROKU_SLUG_ID")

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
				Config: testAccCheckHerokuAppRelease_Basic(appName, slugId),
			},
			{
				ResourceName:      "heroku_app_release.foobar-release",
				ImportStateId:     appName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
