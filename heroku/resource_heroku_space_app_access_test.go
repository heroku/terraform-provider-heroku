package heroku

import (
	"fmt"
	"strings"
	"testing"

	"github.com/heroku/terraform-provider-heroku/v6/helper/test"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccHerokuSpaceAppAccess_Basic(t *testing.T, spaceConfig string) resource.TestStep {
	testUser := testAccConfig.GetNonAdminUserOrAbort(t)

	return resource.TestStep{
		Config: testAccCheckHerokuSpaceAppAccessConfig_basic(spaceConfig, testUser, []string{"create_apps"}),
		Check: resource.ComposeTestCheckFunc(
			test.TestCheckTypeSetElemAttr("heroku_space_app_access.foobar", "permissions.*", "create_apps"),
		),
	}
}

func testAccCheckHerokuSpaceAppAccessConfig_basic(spaceConfig, testUser string, permissions []string) string {
	hclPermissionsList := "[]"
	if len(permissions) > 0 {
		hclPermissionsList = fmt.Sprintf("[\"%s\"]", strings.Join(permissions, "\",\""))
	}
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

resource "heroku_space_app_access" "foobar" {
  space = heroku_space.foobar.name
  email = "%s"
  permissions = %s
}
`, spaceConfig, testUser, hclPermissionsList)
}
