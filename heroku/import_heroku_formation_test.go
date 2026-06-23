package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
