package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHerokuDomain_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDomainConfig_basic(appName),
			},
			{
				ResourceName:        "heroku_domain.one",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
			},
		},
	})
}

func TestAccHerokuDomain_importSSL(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDomainConfig_ssl(appName),
			},
			{
				ResourceName:        "heroku_domain.one",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
			},
		},
	})
}
