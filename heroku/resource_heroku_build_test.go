package heroku

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v6"
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

func TestAccHerokuBuild_Fails(t *testing.T) {
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		ErrorCheck: func(err error) error {
			// Expect the build log output from the Ruby buildpack
			if strings.Contains(err.Error(), "-----> Ruby app detected") {
				return nil
			}
			return err
		},
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuBuildConfig_fails(appName),
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
	// Manually generated using `shasum --algorithm 256 app.tgz`
	// Manually generated using `shasum --algorithm 256 app-2.tgz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums
	sourceChecksum := "SHA256:da57c23d767c971b383de3bf1a680e5ea0f3991f4738552cb383127e60864b20"
	sourceChecksum2 := "SHA256:483332872ad9112337b5790da1406c8b3cdcf07d53d04c953d9e17d3e63fb522"

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

func TestAccHerokuBuild_LocalSourceDirectoryDiff(t *testing.T) {
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

// https://github.com/heroku/terraform-provider-heroku/issues/160
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

		foundBuild, err := client.BuildInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

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
    app_id = heroku_app.foobar.id
    source {
        url = "https://github.com/heroku/terraform-provider-heroku/raw/update-heroku-api-client/heroku/test-fixtures/app.tgz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_fails(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app_id = heroku_app.foobar.id
    source {
        url = "https://github.com/heroku/terraform-provider-heroku/raw/update-heroku-api-client/heroku/test-fixtures/app-broken-build.tgz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_insecureUrl(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app_id = heroku_app.foobar.id
    source {
      url = "http://github.com/mars/terraform-provider-heroku/raw/update-heroku-api-client/heroku/test-fixtures/app.tgz"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_noSource(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app_id = heroku_app.foobar.id
    source {
      version = "v0"
    }
}`, appName)
}

func testAccCheckHerokuBuildConfig_allOpts(appName string) string {
	// Manually generated `checksum` using `shasum --algorithm 256 app.tar.gz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums

	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_build" "foobar" {
    app_id = heroku_app.foobar.id
    buildpacks = [
      "https://github.com/heroku/heroku-buildpack-jvm-common",
      "https://github.com/heroku/heroku-buildpack-ruby",
    ]
    source {
      checksum = "SHA256:da57c23d767c971b383de3bf1a680e5ea0f3991f4738552cb383127e60864b20"
      url = "https://github.com/heroku/terraform-provider-heroku/raw/update-heroku-api-client/heroku/test-fixtures/app.tgz"
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
    app_id = heroku_app.foobar.id
    source {
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
    app_id = heroku_app.foobar.id
    source {
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
    app_id = heroku_app.foobar.id
    source {
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
    app_id = heroku_app.foobar.id
    source {
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
    app_id = heroku_app.foobar.id
    buildpacks = ["https://github.com/heroku/heroku-buildpack-ruby"]
    source {
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
	var renameErr error
	renameErr = os.Rename("test-fixtures/app", "test-fixtures/app-orig")
	renameErr = os.Rename("test-fixtures/app-2", "test-fixtures/app")
	return false, renameErr
}

func resetSourceDirectories() error {
	if _, err := os.Stat("test-fixtures/app-orig"); err == nil {
		var renameErr error
		renameErr = os.Rename("test-fixtures/app", "test-fixtures/app-2")
		renameErr = os.Rename("test-fixtures/app-orig", "test-fixtures/app")
		if renameErr != nil {
			return renameErr
		}
	}
	return nil
}

func switchSourceFiles() (bool, error) {
	var renameErr error
	renameErr = os.Rename("test-fixtures/app.tgz", "test-fixtures/app-orig.tgz")
	renameErr = os.Rename("test-fixtures/app-2.tgz", "test-fixtures/app.tgz")
	return false, renameErr
}

func resetSourceFiles() error {
	if _, err := os.Stat("test-fixtures/app-orig.tgz"); err == nil {
		var renameErr error
		renameErr = os.Rename("test-fixtures/app.tgz", "test-fixtures/app-2.tgz")
		renameErr = os.Rename("test-fixtures/app-orig.tgz", "test-fixtures/app.tgz")
		if renameErr != nil {
			return renameErr
		}
	}
	return nil
}

// Unit tests for build generation functionality
func TestHerokuBuildGeneration(t *testing.T) {
	tests := []struct {
		name       string
		generation string
		feature    string
		expected   bool
	}{
		{
			name:       "Cedar build traditional_buildpacks should be supported",
			generation: "cedar",
			feature:    "traditional_buildpacks",
			expected:   true,
		},
		{
			name:       "Cedar build cloud_native_buildpacks should be unsupported",
			generation: "cedar",
			feature:    "cloud_native_buildpacks",
			expected:   false,
		},
		{
			name:       "Fir build traditional_buildpacks should be unsupported",
			generation: "fir",
			feature:    "traditional_buildpacks",
			expected:   false,
		},
		{
			name:       "Fir build cloud_native_buildpacks should be supported",
			generation: "fir",
			feature:    "cloud_native_buildpacks",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supported := IsFeatureSupported(tt.generation, "build", tt.feature)
			if supported != tt.expected {
				t.Errorf("Expected %t but got %t for generation %s feature %s", tt.expected, supported, tt.generation, tt.feature)
			}
			t.Logf("✅ Generation: %s, Resource: build, Feature: %s, Supported: %t", tt.generation, tt.feature, supported)
		})
	}
}

// Acceptance test for build generation validation
func TestAccHerokuBuild_Generation(t *testing.T) {
	var build heroku.Build
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-build-gen-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Test Cedar generation with buildpacks (should work)
				Config: testAccCheckHerokuBuildConfig_generation(appName, "cedar", true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
					resource.TestCheckResourceAttr("heroku_build.foobar", "generation", "cedar"),
					resource.TestCheckResourceAttr("heroku_build.foobar", "status", "succeeded"),
				),
			},
			{
				// Test Fir generation with buildpacks (should fail during plan)
				Config:      testAccCheckHerokuBuildConfig_generation(appName+"-fir", "fir", true),
				ExpectError: regexp.MustCompile("buildpacks are not supported for fir generation builds"),
			},
			{
				// Test Fir generation without buildpacks (should work)
				Config: testAccCheckHerokuBuildConfig_generation(appName+"-fir-clean", "fir", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuBuildExists("heroku_build.foobar", &build),
					resource.TestCheckResourceAttr("heroku_build.foobar", "generation", "fir"),
					resource.TestCheckResourceAttr("heroku_build.foobar", "status", "succeeded"),
				),
			},
		},
	})
}

func testAccCheckHerokuBuildConfig_generation(appName, generation string, includeBuildpacks bool) string {
	buildpackConfig := ""
	if includeBuildpacks {
		buildpackConfig = `
  buildpacks = ["https://github.com/heroku/heroku-buildpack-nodejs"]`
	}

	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name   = "%s"
  region = "us"
}

resource "heroku_build" "foobar" {
  app_id     = heroku_app.foobar.id
  generation = "%s"%s

  source {
    path = "test-fixtures/app"
  }
}
`, appName, generation, buildpackConfig)
}
