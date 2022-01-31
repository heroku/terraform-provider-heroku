package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccDatasourceHerokuSpacePeeringInfo_Basic(t *testing.T, spaceConfig string) resource.TestStep {
	return resource.TestStep{
		Config: testAccCheckHerokuSpacePeeringInfo_basic(spaceConfig),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"data.heroku_space_peering_info.foobar", "aws_account_id"),
			resource.TestCheckResourceAttr(
				"data.heroku_space_peering_info.foobar", "aws_region", "us-east-1"),
			resource.TestCheckResourceAttrSet(
				"data.heroku_space_peering_info.foobar", "vpc_id"),
			resource.TestCheckResourceAttrSet(
				"data.heroku_space_peering_info.foobar", "vpc_cidr"),
			resource.TestCheckResourceAttrSet(
				"data.heroku_space_peering_info.foobar", "dyno_cidr_blocks.#"),
			resource.TestCheckResourceAttrSet(
				"data.heroku_space_peering_info.foobar", "unavailable_cidr_blocks.#"),
		),
	}
}

func testAccCheckHerokuSpacePeeringInfo_basic(spaceConfig string) string {
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

data "heroku_space_peering_info" "foobar" {
  name = heroku_space.foobar.name
}
`, spaceConfig)
}
