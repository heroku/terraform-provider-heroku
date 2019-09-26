package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuSpaceInboundRuleset_Basic(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space_inbound_ruleset.foobar",
				Config:       testAccCheckHerokuSpaceInboundRulesetConfig_basic(spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_ruleset.foobar", "rule.#", "2"),
				),
			},
		},
	})
}

func testAccCheckHerokuSpaceInboundRulesetConfig_basic(spaceName, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}

resource "heroku_space_inbound_ruleset" "foobar" {
  space = "${heroku_space.foobar.name}"

  rule { 
    action = "allow"
    source = "8.8.8.8/32"
  }

  rule {
    action = "allow"
    source = "8.8.8.0/24"
  }
}
`, spaceName, orgName)
}
