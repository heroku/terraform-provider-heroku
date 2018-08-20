package heroku

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
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
	var slug heroku.Slug
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSlugConfig_withFile(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSlugExists("heroku_slug.foobar", &slug),
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
    stack = "heroku-18"
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
    stack = "heroku-18"
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
	wd, _ := os.Getwd()
	filePath := wd + "/test-fixtures/slug.tgz"

	file, _ := os.Open(filePath)
	defer file.Close()

	hash := sha256.New()
	io.Copy(hash, file)
	fileChecksum := hash.Sum(nil)

	return fmt.Sprintf(`resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
    stack = "heroku-18"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    buildpack_provided_description = "Ruby"
    file_path = "%s"
    checksum = "%x"
    process_types = {
    	web = "ruby server.rb"
    }
}`, appName, filePath, fileChecksum)
}
