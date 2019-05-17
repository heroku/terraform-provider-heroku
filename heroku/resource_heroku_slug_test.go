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
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuSlug_Basic(t *testing.T) {
	var slug heroku.Slug
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug),
				),
			},
		},
	})
}

func TestAccHerokuSlug_NoFile(t *testing.T) {
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuSlugConfig_noFile(appName),
				ExpectError: regexp.MustCompile(`requires either`),
			},
		},
	})
}

func TestAccHerokuSlug_AllOpts(t *testing.T) {
	var slug heroku.Slug
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_allOpts(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug),
				),
			},
		},
	})
}

func TestAccHerokuSlug_WithFile(t *testing.T) {
	var slug, slug2 heroku.Slug
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)
	// Manually generated using `shasum --algorithm 256 slug.tgz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums
	slugChecksum := "SHA256:6731cb5caea2cda97c6177216373360a0733aa8e7a21801de879fa8d22f740cf"
	slugChecksum2 := "SHA256:61fb0a7b414a94c316b8284680fe5538e2b8e6ed3989e4f1ee52817fb72ea10c"

	defer resetSlugFiles()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_withFile(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug),
					resource.TestCheckResourceAttr("heroku_slug.foobar", "checksum", slugChecksum),
				),
			},
			{
				SkipFunc: switchSlugFiles,
				Config:   testAccCheckHerokuSlugConfig_withFile(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug2),
					resource.TestCheckResourceAttr("heroku_slug.foobar", "checksum", slugChecksum2),
				),
			},
		},
	})
}

func TestAccHerokuSlug_WithRemoteFile(t *testing.T) {
	var slug heroku.Slug
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)
	// Manually generated using `shasum --algorithm 256 slug.tgz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums
	slugChecksum := "SHA256:6731cb5caea2cda97c6177216373360a0733aa8e7a21801de879fa8d22f740cf"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_withRemoteFile(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug),
					resource.TestCheckResourceAttr("heroku_slug.foobar", "checksum", slugChecksum),
				),
			},
		},
	})
}

func TestAccHerokuSlug_WithInsecureRemoteFile(t *testing.T) {
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuSlugConfig_withInsecureRemoteFile(appName),
				ExpectError: regexp.MustCompile(`must be a secure URL`),
			},
		},
	})
}

func TestAccHerokuSlug_WithFile_InPrivateSpace(t *testing.T) {
	var slug heroku.Slug
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)
	orgName := testAccConfig.GetSpaceOrganizationOrSkip(t)
	spaceName := fmt.Sprintf("tftest-%s", randString)
	// Manually generated using `shasum --algorithm 256 slug.tgz`
	// per Heroku docs https://devcenter.heroku.com/articles/slug-checksums
	slugChecksum := "SHA256:6731cb5caea2cda97c6177216373360a0733aa8e7a21801de879fa8d22f740cf"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_withFile_inPrivateSpace(spaceName, orgName, appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug),
					resource.TestCheckResourceAttr("heroku_slug.foobar", "checksum", slugChecksum),
				),
			},
		},
	})
}

func testAccCheckHerokuSlugExists(n string, Slug *heroku.Slug) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Slug ID is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundSlug, err := client.SlugInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundSlug.ID != rs.Primary.ID {
			return fmt.Errorf("Slug not found")
		}

		*Slug = *foundSlug

		return nil
	}
}

func testAccCheckHerokuSlugConfig_basic(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    file_path = "test-fixtures/slug.tgz"
    process_types = {
    	test = "echo 'Just a test'"
    	diag = "echo 'Just diagnosis'"
    }
}`, appName)
}

func testAccCheckHerokuSlugConfig_noFile(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    process_types = {
      test = "echo 'Just a test'"
      diag = "echo 'Just diagnosis'"
    }
}`, appName)
}

func testAccCheckHerokuSlugConfig_allOpts(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpack_provided_description = "Test Language"
    file_path = "test-fixtures/slug.tgz"
    commit = "abcde"
    commit_description = "Build for testing"
    process_types = {
    	test = "echo 'Just a test'"
    	diag = "echo 'Just diagnosis'"
    }
    stack = "heroku-18"
}`, appName)
}

func testAccCheckHerokuSlugConfig_withFile(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpack_provided_description = "Ruby"
    file_path = "test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}`, appName)
}

func testAccCheckHerokuSlugConfig_withRemoteFile(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpack_provided_description = "Ruby"
    file_url = "https://github.com/terraform-providers/terraform-provider-heroku/raw/master/heroku/test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}`, appName)
}

func testAccCheckHerokuSlugConfig_withInsecureRemoteFile(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpack_provided_description = "Ruby"
    file_url = "http://github.com/terraform-providers/terraform-provider-heroku/raw/master/heroku/test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}`, appName)
}

func testAccCheckHerokuSlugConfig_withFile_inPrivateSpace(spaceName, orgName, appName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
  region = "virginia"
}

resource "heroku_app" "foobar" {
  name = "%s"
  space = "${heroku_space.foobar.name}"
  region = "virginia"

  organization {
    name = "%s"
  }
}

resource "heroku_slug" "foobar" {
  app = "${heroku_app.foobar.name}"
  buildpack_provided_description = "Ruby"
  file_path = "test-fixtures/slug.tgz"
  process_types = {
    web = "ruby server.rb"
  }
}`, spaceName, orgName, appName, orgName)
}

func switchSlugFiles() (bool, error) {
	os.Rename("test-fixtures/slug.tgz", "test-fixtures/slug-orig.tgz")
	os.Rename("test-fixtures/slug-2.tgz", "test-fixtures/slug.tgz")
	return false, nil
}

func resetSlugFiles() error {
	if _, err := os.Stat("test-fixtures/slug-orig.tgz"); err == nil {
		os.Rename("test-fixtures/slug.tgz", "test-fixtures/slug-2.tgz")
		os.Rename("test-fixtures/slug-orig.tgz", "test-fixtures/slug.tgz")
	}
	return nil
}
