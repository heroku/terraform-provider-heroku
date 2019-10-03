package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuSpaceAppAccess_importBasic(t *testing.T) {
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)
	testUser := testAccConfig.GetNonAdminUserOrAbort(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSpaceAppAccessConfig_basic(spaceName, org, testUser, []string{"create_apps"}),
			},
			{
				ResourceName:      "heroku_space_app_access.foobar",
				ImportStateId:     buildCompositeID(spaceName, testUser),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
