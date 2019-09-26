package heroku

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"regexp"
	"testing"
)

func TestAccHerokuConfig_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfig_Single(),
			},
			{
				ResourceName:      "heroku_config.foobar",
				ImportState:       true,
				ImportStateVerify: true,
				ExpectError:       regexp.MustCompile(`not possible to import heroku_config`),
			},
		},
	})
}
