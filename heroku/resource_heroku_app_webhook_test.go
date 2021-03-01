package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuAppWebhook_Basic(t *testing.T) {
	var webhook heroku.AppWebhookInfoResult
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAppWebhookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppWebhookConfig(appName, "https://terraform.example.com:1234", "notify", "api:release"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppWebhookExists("heroku_app_webhook.foobar_webhook", &webhook),
					testAccCheckHerokuAppWebhookAttributes(&webhook, "https://terraform.example.com:1234", "notify", "api:release"),
					resource.TestCheckResourceAttr("heroku_app_webhook.foobar_webhook", "url", "https://terraform.example.com:1234"),
					resource.TestCheckResourceAttr("heroku_app_webhook.foobar_webhook", "level", "notify"),
					resource.TestCheckResourceAttr("heroku_app_webhook.foobar_webhook", "include.0", "api:release"),
				),
			},
			{
				Config: testAccCheckHerokuAppWebhookConfig(appName, "https://terraform.example.com:4321", "sync", "api:build"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAppWebhookExists("heroku_app_webhook.foobar_webhook", &webhook),
					testAccCheckHerokuAppWebhookAttributes(&webhook, "https://terraform.example.com:4321", "sync", "api:build"),
					resource.TestCheckResourceAttr("heroku_app_webhook.foobar_webhook", "url", "https://terraform.example.com:4321"),
					resource.TestCheckResourceAttr("heroku_app_webhook.foobar_webhook", "level", "sync"),
					resource.TestCheckResourceAttr("heroku_app_webhook.foobar_webhook", "include.0", "api:build"),
				),
			},
		},
	})
}

func testAccCheckHerokuAppWebhookConfig(appName, url, level, include string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
    name = "%s"
    region = "us"
}

resource "heroku_app_webhook" "foobar_webhook" {
    app_id  = "${heroku_app.foobar.id}"
    url     = "%s"
    level   = "%s"
    include = ["%s"]
}`, appName, url, level, include)
}

func testAccCheckHerokuAppWebhookDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_app_webhook" {
			continue
		}

		_, err := client.AppWebhookInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Webhook still exists")
		}
	}

	return nil
}

func testAccCheckHerokuAppWebhookExists(n string, Webhook *heroku.AppWebhookInfoResult) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Webhook ID is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundWebhook, err := client.AppWebhookInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundWebhook.ID != rs.Primary.ID {
			return fmt.Errorf("Webhook not found")
		}

		*Webhook = *foundWebhook

		return nil
	}
}

func testAccCheckHerokuAppWebhookAttributes(Webhook *heroku.AppWebhookInfoResult, url, level, include string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if Webhook.URL != url {
			return fmt.Errorf("Bad URL: %s", Webhook.URL)
		}

		if Webhook.Level != level {
			return fmt.Errorf("Bad Level: %s", Webhook.Level)
		}

		if len(Webhook.Include) != 1 || Webhook.Include[0] != include {
			return fmt.Errorf("Bad Include: %v", Webhook.Include)
		}

		return nil
	}
}
