package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"regexp"
	"testing"
)

func TestAccHerokuConfig_Single(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfig_Single(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuConfigExists("heroku_config.foobar"),
					resource.TestCheckResourceAttr(
						"heroku_config.foobar", "vars.RAILS_ENV", "PROD"),
					resource.TestCheckResourceAttr(
						"heroku_config.foobar", "vars.LOG_LEVEL", "DEBUG"),
				),
			},
		},
	})
}

func TestAccHerokuConfig_Both(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuConfig_Both(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuConfigExists("heroku_config.foobar"),
					resource.TestCheckResourceAttr(
						"heroku_config.foobar", "vars.RAILS_ENV", "PROD"),
					resource.TestCheckResourceAttr(
						"heroku_config.foobar", "sensitive_vars.PRIVATE_KEY", "it_is_a_secret"),
				),
			},
		},
	})
}

func TestAccHerokuConfig_Dupe(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckHerokuConfig_Dupe(name),
				ExpectError: regexp.MustCompile(`duplicate config vars`),
			},
		},
	})
}

func testAccCheckHerokuConfigExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("config not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No config ID set")
		}

		return nil
	}
}

func testAccCheckHerokuConfig_Single(name string) string {
	return fmt.Sprintf(`
resource "heroku_config" "foobar" {
    name = "%s"

    vars = {
       RAILS_ENV = "PROD"
       LOG_LEVEL = "DEBUG"
    }
}
`, name)
}

func testAccCheckHerokuConfig_Both(name string) string {
	return fmt.Sprintf(`
resource "heroku_config" "foobar" {
    name = "%s"

    vars = {
       RAILS_ENV = "PROD"
       LOG_LEVEL = "DEBUG"
    }

    sensitive_vars = {
        PRIVATE_KEY = "it_is_a_secret"
        API_TOKEN   = "some_token"
    }
}
`, name)
}

func testAccCheckHerokuConfig_Dupe(name string) string {
	return fmt.Sprintf(`
resource "heroku_config" "foobar" {
    name = "%s"

    vars = {
       RAILS_ENV = "PROD"
       PRIVATE_KEY = "it_is_a_secret"
    }

    sensitive_vars = {
        PRIVATE_KEY = "it_is_a_secret"
        API_TOKEN   = "some_token"
    }
}
`, name)
}
