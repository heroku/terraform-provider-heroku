package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuAppWebhook_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAppWebhookConfig(appName, "https://terraform.example.com:4321", "sync", "api:build"),
			},
			{
				ResourceName:        "heroku_app_webhook.foobar_webhook",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
			},
		},
	})
}
