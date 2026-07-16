package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccHerokuAddon_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_basic(appName, org),
			},
			{
				ResourceName:            "heroku_addon.foobar",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_vars", "config"},
			},
			{
				Config:             testAccCheckHerokuAddonConfig_basic(appName, org),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
