package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccHerokuSpaceInboundRuleset_Basic(t *testing.T, spaceConfig string) resource.TestStep {
	return resource.TestStep{
		ResourceName: "heroku_space_inbound_ruleset.foobar",
		Config:       testAccCheckHerokuSpaceInboundRulesetConfig_basic(spaceConfig),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"heroku_space_inbound_ruleset.foobar", "rule.#", "2"),
		),
	}
}

func testAccCheckHerokuSpaceInboundRulesetConfig_basic(spaceConfig string) string {
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

resource "heroku_space_inbound_ruleset" "foobar" {
  space = heroku_space.foobar.name

  rule { 
    action = "allow"
    source = "8.8.8.8/32"
  }

  rule {
    action = "allow"
    source = "8.8.8.0/24"
  }
}
`, spaceConfig)
}
