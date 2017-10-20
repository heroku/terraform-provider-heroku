package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuAddonAttachment_Basic(t *testing.T) {
	appName := fmt.Sprintf("apptest-%s", acctest.RandString(10))
	addonName := fmt.Sprintf("addontest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonAttachmentConfig_basic(appName, addonName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_addon_attachment.foobar", appName, addonName),
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
