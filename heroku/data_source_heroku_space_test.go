package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccDatasourceHerokuSpace_Basic(t *testing.T, spaceConfig string) resource.TestStep {
	orgName := testAccConfig.GetSpaceOrganizationOrSkip(t)

	return resource.TestStep{
		Config: testAccCheckHerokuSpaceWithDatasource_basic(spaceConfig),
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.heroku_space.foobar", "organization", orgName),
			resource.TestCheckResourceAttr(
				"data.heroku_space.foobar", "region", "virginia"),
		),
	}
}

func testAccCheckHerokuSpaceWithDatasource_basic(spaceConfig string) string {
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

data "heroku_space" "foobar" {
  name = heroku_space.foobar.name
}
`, spaceConfig)
}
