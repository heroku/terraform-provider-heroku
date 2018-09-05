package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuSlug_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_basic(appName),
			},
			{
				ResourceName:        "heroku_slug.foobar",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
				ImportStateVerify:   true,
				// "blob" ignored because generated uniquely by Heroku for each Slug
				ImportStateVerifyIgnore: []string{"blob"},
			},
		},
	})
}

func TestAccHerokuSlug_importAllOpts(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_allOpts(appName),
			},
			{
				ResourceName:        "heroku_slug.foobar",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
				ImportStateVerify:   true,
				// "blob" ignored because generated uniquely by Heroku for each Slug
				ImportStateVerifyIgnore: []string{"blob"},
			},
		},
	})
}

func TestAccHerokuSlug_importWithFile(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_withFile(appName),
			},
			{
				ResourceName:        "heroku_slug.foobar",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
				ImportStateVerify:   true,
				// "blob" ignored because generated uniquely by Heroku for each Slug
				// "file_path" ignored because it is an ephemeral create-only attribute
				ImportStateVerifyIgnore: []string{"blob", "file_path"},
			},
		},
	})
}
