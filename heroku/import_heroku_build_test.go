package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuBuild_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_basic(appName),
			},
			{
				ResourceName:            "heroku_build.foobar",
				ImportStateIdPrefix:     appName + ":",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"local_checksum", "source", "output_stream_url", "status"},
			},
		},
	})
}

func TestAccHerokuBuild_importAllOpts(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_allOpts(appName),
			},
			{
				ResourceName:            "heroku_build.foobar",
				ImportStateIdPrefix:     appName + ":",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"local_checksum", "source", "output_stream_url", "status"},
			},
		},
	})
}

func TestAccHerokuBuild_importWithFileUrl(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_localSource(appName),
			},
			{
				ResourceName:            "heroku_build.foobar",
				ImportStateIdPrefix:     appName + ":",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"local_checksum", "source", "output_stream_url", "status"},
			},
		},
	})
}
