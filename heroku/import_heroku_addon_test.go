package heroku

import (
	"fmt"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuAddon_importBasic(t *testing.T) {
	var addon heroku.AddOn
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuAddonDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAddonConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAddonExists("heroku_addon.foobar", &addon),
					testAccCheckHerokuAddonAttributes(&addon, "deployhooks:http"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "config.0.url", "http://google.com"),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "app", appName),
					resource.TestCheckResourceAttr(
						"heroku_addon.foobar", "plan", "deployhooks:http"),
				),
			},
			{
				ResourceName:            "heroku_addon.foobar",
				ImportState:             true,
				ImportStateCheck:        testAccCheckHerokuAddonImportedAttributes,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_vars"},
			},
		},
	})
}

func testAccCheckHerokuAddonImportedAttributes(state []*terraform.InstanceState) error {
	for _, s := range state {
		fmt.Printf("%+v\n", s)
		if app, ok := s.Attributes["app"]; !ok || app == "" {
			return fmt.Errorf("No Addon app is set")
		}
	}
	return nil
}
