package heroku

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccHerokuSpaceAppAccess_importBasic(t *testing.T, spaceName string) resource.TestStep {
	testUser := testAccConfig.GetNonAdminUserOrAbort(t)

	return resource.TestStep{
		ResourceName:      "heroku_space_app_access.foobar",
		ImportStateId:     buildCompositeID(spaceName, testUser),
		ImportState:       true,
		ImportStateVerify: true,
	}
}
