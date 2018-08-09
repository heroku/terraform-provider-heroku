package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuFormation_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	var slugID string
	var org string
	formationType := "web"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			org = testAccConfig.GetOrganizationOrSkip(t)
			slugID = testAccConfig.GetSlugIDOrSkip(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuFormationConfig_WithOrg(org, appName, slugID, "standard-2x", 2),
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
