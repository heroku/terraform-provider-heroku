package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuFormation_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)
	slugID := testAccConfig.GetSlugIDOrSkip(t)
	formationType := "web"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
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
