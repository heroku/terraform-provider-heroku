package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"regexp"
	"testing"
)

func TestAccHerokuConfigAssociation_importBasic(t *testing.T) {
	org := testAccConfig.GetOrganizationOrSkip(t)
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfigAssociation_Basic(org, name),
			},
			{
				ResourceName:      "heroku_config_association.foobar-config",
				ImportState:       true,
				ImportStateVerify: true,
				ExpectError:       regexp.MustCompile(`not possible to import`),
			},
		},
	})
}
