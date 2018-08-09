package heroku

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuAppRelease_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	var slugID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			slugID = testAccConfig.GetSlugIDOrSkip(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppRelease_Basic(appName, slugID),
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
