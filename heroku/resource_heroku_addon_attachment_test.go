package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccHerokuAddonAttachment_Basic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_basic(appName, org),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"heroku_addon_attachment.foobar", "app_id"),
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "namespace", "TEST_NAMESPACE"),
				),
			},
		},
	})
}

func TestAccHerokuAddonAttachment_Named(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	org := testAccConfig.GetOrganizationOrSkip(t)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_named(appName, org, "TEST_ADDON"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"heroku_addon_attachment.foobar", "app_id"),
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", "name", "TEST_ADDON"),
				),
			},
		},
	})
}

func testAccCheckHerokuAddonAttachmentConfig_basic(appID, org string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name   = "%s"
	region = "us"

	organization {
		name = "%s"
	}
}

resource "heroku_addon" "foobar" {
    app_id = heroku_app.foobar.id
    plan = "heroku-postgresql"
}

resource "heroku_addon_attachment" "foobar" {
    app_id    = heroku_app.foobar.id
    addon_id  = heroku_addon.foobar.id
    namespace = "TEST_NAMESPACE"
}`, appID, org)
}

func testAccCheckHerokuAddonAttachmentConfig_named(appID, org, name string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
	name   = "%s"
	region = "us"

	organization {
		name = "%s"
	}
}

resource "heroku_addon" "foobar" {
    app_id = heroku_app.foobar.id
    plan = "heroku-postgresql"
}

resource "heroku_addon_attachment" "foobar" {
    app_id   = heroku_app.foobar.id
    addon_id = heroku_addon.foobar.id
    name     = "%s"
}`, appID, org, name)
}
