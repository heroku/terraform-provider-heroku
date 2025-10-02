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
			resource.TestCheckResourceAttr(
				"data.heroku_space.foobar", "generation", "cedar"),
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

// testStep_AccDatasourceHerokuSpace_Generation_Fir tests the space data source with a Fir space
func testStep_AccDatasourceHerokuSpace_Generation_Fir(t *testing.T, spaceConfig string) resource.TestStep {
	orgName := testAccConfig.GetAnyOrganizationOrSkip(t)

	config := fmt.Sprintf(`%s

data "heroku_space" "data_source_test" {
  name = heroku_space.foobar.name
}`, spaceConfig)

	return resource.TestStep{
		Config: config,
		Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.heroku_space.data_source_test", "name"),
			resource.TestCheckResourceAttr("data.heroku_space.data_source_test", "generation", "fir"),
			resource.TestCheckResourceAttr("data.heroku_space.data_source_test", "organization", orgName),
			resource.TestCheckResourceAttr("data.heroku_space.data_source_test", "shield", "false"),
			resource.TestCheckResourceAttrSet("data.heroku_space.data_source_test", "id"),
			resource.TestCheckResourceAttrSet("data.heroku_space.data_source_test", "uuid"),
			resource.TestCheckResourceAttrSet("data.heroku_space.data_source_test", "region"),
		),
	}
}
