package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuDrain_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDrainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDrainConfig_basic(appName),
			},
			{
				ResourceName:        "heroku_drain.foobar",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
			},
		},
	})
}
