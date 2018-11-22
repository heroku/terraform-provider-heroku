package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccHerokuAppConfigVar_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfigVar_Basic(appName),
			},
			{
				ResourceName:      "heroku_app_config_var.foobar-configs",
				ImportStateId:     fmt.Sprintf("%s:%s:%s", appName, "public", "ENVIRONMENT,USER"),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
