package heroku

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	heroku "github.com/heroku/heroku-go/v3"
)

func TestAccHerokuSpaceAppAccess_Basic(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := getTestSpaceOrganizationName()
	testUser := getTestUser()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccSkipTestIfOrganizationMissing(t)
			testAccSkipTestIfUserMissing(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSpaceAppAccessConfig_basic(spaceName, org, testUser, []string{"create_apps"}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space_app_access.foobar", "permissions.3695762012", "create_apps"),
				),
			},
		},
	})
}

func testAccCheckHerokuSpaceAppAccessConfig_basic(spaceName, orgName, testUser string, permissions []string) string {
	hclPermissionsList := "[]"
	if len(permissions) > 0 {
		hclPermissionsList = fmt.Sprintf("[\"%s\"]", strings.Join(permissions, "\",\""))
	}
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}

resource "heroku_space_app_access" "foobar" {
  space = "${heroku_space.foobar.name}"
  email = "%s"
  permissions = %s
}
`, spaceName, orgName, testUser, hclPermissionsList)
}
