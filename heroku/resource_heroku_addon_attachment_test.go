package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuAddonAttachment_Basic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "app_id", appName),
				),
			},
		},
	})
}

func TestAccHerokuAddonAttachment_Named(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_named(appName, "TEST_ADDON"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "app_id", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "name", "TEST_ADDON"),
				),
			},
		},
	})
}

func testAccCheckHerokuAddonAttachmentConfig_basic(appName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name   = "%s"
	region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "heroku-postgresql:hobby-dev"
}

resource "heroku_addon_attachment" "foobar" {
    app_id   = "${heroku_app.foobar.id}"
    addon_id = "${heroku_addon.foobar.id}"
}`, appName)
}

func testAccCheckHerokuAddonAttachmentConfig_named(appName string, name string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name   = "%s"
	region = "us"
}

resource "heroku_addon" "foobar" {
    app = "${heroku_app.foobar.name}"
    plan = "heroku-postgresql:hobby-dev"
}

resource "heroku_addon_attachment" "foobar" {
    app_id   = "${heroku_app.foobar.id}"
    addon_id = "${heroku_addon.foobar.id}"
    name     = "%s"
}`, appName, name)
}
