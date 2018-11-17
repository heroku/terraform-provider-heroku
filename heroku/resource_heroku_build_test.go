package heroku

import (
	"context"
	"fmt"
	"regexp"
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

func TestAccHerokuBuild_InsecureUrl(t *testing.T) {
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuBuildConfig_insecureUrl(appName),
				ExpectError: regexp.MustCompile(`must be a secure URL`),
			},
		},
	})
}

func TestAccHerokuBuild_AllOpts(t *testing.T) {
	var build heroku.Build
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_allOpts(appName),
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

func testAccCheckHerokuBuildConfig_insecureUrl(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
      url = "http://github.com/mars/cra-example-app/archive/v2.1.1.tar.gz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_allOpts(appName string) string {
	// Manually generated `checksum` using `shasum --algorithm 256 v2.1.1.tar.gz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums

	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpacks = ["mars/create-react-app"]
    source = {
      checksum = "SHA256:b7dfb201c9fa6541b64fd450c5e00641c80d7d1e39134b7c12ce601efbb8642b"
      url = "https://github.com/mars/cra-example-app/archive/v2.1.1.tar.gz"
      version = "v2.1.1"
    }
}`, appName)
}
