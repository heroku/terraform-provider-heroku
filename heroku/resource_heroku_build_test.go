package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/heroku-go/v3"
)

func TestAccHerokuBuild_Basic(t *testing.T) {
	var build heroku.Build
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
				),
			},
		},
	})
}

func testAccCheckHerokuBuildExists(n string, Build *heroku.Build) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Build ID is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundBuild, err := client.BuildInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundBuild.ID != rs.Primary.ID {
			return fmt.Errorf("Build not found")
		}

		*Build = *foundBuild

		return nil
	}
}

func testAccCheckHerokuBuildConfig_basic(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
    	url = "https://github.com/mars/cra-example-app/archive/v2.1.1.tar.gz"
    }
}`, appName)
}
