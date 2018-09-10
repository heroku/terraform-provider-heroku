package heroku

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/heroku-go/v3"
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

func testAccCheckHerokuSlugExists(n string, Slug *heroku.Slug) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Slug ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

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
    checksum = "54321"
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
