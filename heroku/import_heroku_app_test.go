package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuApp_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	appStack := "heroku-16"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_basic(appName, appStack),
			},
			{
				ResourceName:      "heroku_app.foobar",
				ImportState:       true,
				ImportStateVerify: true,

				// Due to the nature of these two attributes, it will not be possible to import them as part of the resource import.
				ImportStateVerifyIgnore: []string{"config_vars", "sensitive_config_vars"},
			},
		},
	})
}

func TestAccHerokuApp_importOrganization(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_organization(appName, org),
			},
			{
				ResourceName:      "heroku_app.foobar",
				ImportState:       true,
				ImportStateVerify: true,

				// Due to the nature of these two attributes, it will not be possible to import them as part of the resource import.
				ImportStateVerifyIgnore: []string{"config_vars", "sensitive_config_vars"},
			},
		},
	})
}

func TestAccHerokuApp_importBuildpacks(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppConfig_multi(appName),
			},
			{
				ResourceName:      "heroku_app.foobar",
				ImportState:       true,
				ImportStateVerify: true,

				// Due to the nature of these two attributes, it will not be possible to import them as part of the resource import.
				ImportStateVerifyIgnore: []string{"config_vars", "sensitive_config_vars"},
			},
		},
	})
}
