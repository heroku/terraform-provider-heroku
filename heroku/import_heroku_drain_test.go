package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
				ImportStateVerify:   true,
			},
		},
	})
}

func TestAccHerokuDrain_importBasicWithSensitiveURL(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDrainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDrainConfig_basicWithSensitiveURL(appName),
			},
			{
				ResourceName:      "heroku_drain.foobar",
				ImportStateIdFunc: testAccHerokuDrainImportStateIDFunc("heroku_drain.foobar"),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccHerokuDrainImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("[ERROR] Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s:%s:sensitive", rs.Primary.Attributes["app_id"], rs.Primary.Attributes["id"]), nil
	}
}
