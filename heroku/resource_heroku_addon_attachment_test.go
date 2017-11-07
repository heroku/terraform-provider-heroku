package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuAddonAttachment_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_basic("test-addon-12345", "test-heroku-app"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "addon_id", "test-addon-12345"),
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "app_id", "test-heroku-app"),
				),
			},
		},
	})
}

func TestAccHerokuAddonAttachment_Named(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_named("test-addon-67890", "test-heroku-app", "TEST_ADDON"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "addon_id", "test-addon-67890"),
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "app_id", "test-heroku-app"),
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "name", "TEST_ADDON"),
				),
			},
		},
	})
}

func testAccCheckHerokuAddonAttachmentConfig_basic(appName string, addonName string) string {
	return fmt.Sprintf(`
resource "heroku_addon_attachment" "foobar" {
    app_id   = "%s"
    addon_id = "%s"
}`, appName, addonName)
}

func testAccCheckHerokuAddonAttachmentConfig_named(appName string, addonName string, name string) string {
	return fmt.Sprintf(`
resource "heroku_addon_attachment" "foobar" {
    app_id   = "%s"
    addon_id = "%s"
    name     = "%s"
}`, appName, addonName, name)
}
