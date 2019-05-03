package heroku

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/heroku-go"
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
					resource.TestCheckResourceAttr("heroku_build.foobar", "status", "succeeded"),
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

func TestAccHerokuBuild_NoSource(t *testing.T) {
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuBuildConfig_noSource(appName),
				ExpectError: regexp.MustCompile(`Build requires either`),
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

func TestAccHerokuBuild_LocalSourceTarball(t *testing.T) {
	var build, build2 heroku.Build
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)
	// Manually generated using `shasum --algorithm 256 slug.tgz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums
	sourceChecksum := "SHA256:14671a3dcf1ba3f4976438bfd4654da5d2b18ccefa59d10187ecc1286f08ee29"
	sourceChecksum2 := "SHA256:a60dabd2ab4253e85a1a13734dcc444e830f61995247cd307655219c2504738a"

	defer resetSourceFiles()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_localSourceTarball(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
					resource.TestCheckResourceAttr("heroku_build.foobar", "local_checksum", sourceChecksum),
				),
			},
			{
				SkipFunc: switchSourceFiles,
				Config:   testAccCheckHerokuBuildConfig_localSourceTarball(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build2),
					resource.TestCheckResourceAttr("heroku_build.foobar", "local_checksum", sourceChecksum2),
				),
			},
		},
	})
}

func TestAccHerokuBuild_LocalSourceTarball_SetChecksum(t *testing.T) {
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuBuildConfig_localSourceTarball_setChecksum(appName),
				ExpectError: regexp.MustCompile(`checksum should be empty`),
			},
		},
	})
}

func TestAccHerokuBuild_LocalSourceTarball_AllOpts(t *testing.T) {
	var build heroku.Build
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_localSourceTarball_allOpts(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
				),
			},
		},
	})
}

func TestAccHerokuBuild_LocalSourceDirectory(t *testing.T) {
	var build, build2 heroku.Build
	var originalSourceChecksum string
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	defer resetSourceDirectories()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_localSourceDirectory(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
					testAccCheckCaptureSourceChecksum("heroku_build.foobar", &originalSourceChecksum),
				),
			},
			{
				SkipFunc: switchSourceDirectories,
				Config:   testAccCheckHerokuBuildConfig_localSourceDirectory(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build2),
					testAccCheckSourceChecksumIsDifferent("heroku_build.foobar", &originalSourceChecksum),
				),
			},
		},
	})
}

// https://github.com/terraform-providers/terraform-provider-heroku/issues/160
func TestAccHerokuBuild_LocalSourceDirectorySelfContained(t *testing.T) {
	var build heroku.Build
	defer func() { _ = resetSourceDirectories() }()

	// cd to the ./test-fixtures/app directory before and revert back afterwards
	_ = os.Chdir("./test-fixtures/app")
	defer func() { _ = os.Chdir("../..") }()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_localSourceDirectorySelfContained(fmt.Sprintf("tftest-%s", acctest.RandString(10))),
				Check:  testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
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
    	url = "https://github.com/mars/terraform-provider-heroku/raw/build-resource/heroku/test-fixtures/app.tgz"
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
      url = "http://github.com/mars/terraform-provider-heroku/raw/build-resource/heroku/test-fixtures/app.tgz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_noSource(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
      version = "v0"
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
    buildpacks = [
      "https://github.com/heroku/heroku-buildpack-jvm-common",
      "https://github.com/heroku/heroku-buildpack-ruby",
    ]
    source = {
      checksum = "SHA256:14671a3dcf1ba3f4976438bfd4654da5d2b18ccefa59d10187ecc1286f08ee29"
      url = "https://github.com/mars/terraform-provider-heroku/raw/build-resource/heroku/test-fixtures/app.tgz"
      version = "v0"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_localSourceDirectory(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
      path = "test-fixtures/app/"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_localSourceDirectorySelfContained(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}
 resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
      path = "."
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_localSourceTarball(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
      path = "test-fixtures/app.tgz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_localSourceTarball_setChecksum(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    source = {
      checksum = "SHA256:0000000000000000000000000000000000000000000000000000000000000000"
      path = "test-fixtures/app.tgz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_localSourceTarball_allOpts(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpacks = ["https://github.com/heroku/heroku-buildpack-ruby"]
    source = {
      path = "test-fixtures/app.tgz"
      version = "v0"
    }
}`, appName)
}

func testAccCheckCaptureSourceChecksum(buildName string, originalSourceChecksum *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[buildName]
		if !ok {
			return fmt.Errorf("Not found: %s", buildName)
		}
		*originalSourceChecksum = rs.Primary.Attributes["local_checksum"]
		return nil
	}
}

func testAccCheckSourceChecksumIsDifferent(buildName string, originalSourceChecksum *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[buildName]
		if !ok {
			return fmt.Errorf("Not found: %s", buildName)
		}
		if rs.Primary.Attributes["local_checksum"] == *originalSourceChecksum {
			return fmt.Errorf("Checksum should be different")
		}
		return nil
	}
}

func switchSourceDirectories() (bool, error) {
	os.Rename("test-fixtures/app", "test-fixtures/app-orig")
	os.Rename("test-fixtures/app-2", "test-fixtures/app")
	return false, nil
}

func resetSourceDirectories() error {
	if _, err := os.Stat("test-fixtures/app-orig"); err == nil {
		os.Rename("test-fixtures/app", "test-fixtures/app-2")
		os.Rename("test-fixtures/app-orig", "test-fixtures/app")
	}
	return nil
}

func switchSourceFiles() (bool, error) {
	os.Rename("test-fixtures/app.tgz", "test-fixtures/app-orig.tgz")
	os.Rename("test-fixtures/app-2.tgz", "test-fixtures/app.tgz")
	return false, nil
}

func resetSourceFiles() error {
	if _, err := os.Stat("test-fixtures/app-orig.tgz"); err == nil {
		os.Rename("test-fixtures/app.tgz", "test-fixtures/app-2.tgz")
		os.Rename("test-fixtures/app-orig.tgz", "test-fixtures/app.tgz")
	}
	return nil
}
