package heroku

import (
	"context"
	"fmt"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuSpaceInboundRuleset_Basic(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := getTestingOrgName()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
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
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_ruleset.foobar", "rule.1.action", "allow"),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_ruleset.foobar", "rule.1.source", "8.8.8.8/32"),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_ruleset.foobar", "rule.2.action", "allow"),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_ruleset.foobar", "rule.2.source", "8.8.8.0/24"),
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
    space = "${heroku_space.foobar.name}

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

func testAccCheckHerokuSpaceInboundRulesetIsSet(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No space name set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundSpace, err := client.SpaceInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundSpace.ID != rs.Primary.ID {
			return fmt.Errorf("Space not found")
		}

		return nil
	}
}
