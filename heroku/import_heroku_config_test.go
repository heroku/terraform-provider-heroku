package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"regexp"
	"testing"
)

func TestAccHerokuConfig_importBasic(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfig_Single(name),
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
