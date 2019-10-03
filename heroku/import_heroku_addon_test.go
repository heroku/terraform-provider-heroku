package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuAddon_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_basic(appName),
			},
			{
				ResourceName:            "heroku_addon.foobar",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_vars", "config"},
			},
			{
				Config:             testAccCheckHerokuAddonConfig_basic(appName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
