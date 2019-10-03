package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuDomain_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDomainConfig_basic(appName),
			},
			{
				ResourceName:        "heroku_domain.foobar",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
			},
		},
	})
}
