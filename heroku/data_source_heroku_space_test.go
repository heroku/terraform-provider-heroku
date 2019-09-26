package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDatasourceHerokuSpace_Basic(t *testing.T) {
	spaceName := fmt.Sprintf("tftest-space-%s", acctest.RandString(10))
	orgName := testAccConfig.GetSpaceOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSpace_basic(spaceName, orgName),
			},
			{
				Config: testAccCheckHerokuSpaceWithDatasource_basic(spaceName, orgName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.heroku_space.foobar", "name", spaceName),
					resource.TestCheckResourceAttr(
						"data.heroku_space.foobar", "organization", orgName),
					resource.TestCheckResourceAttr(
						"data.heroku_space.foobar", "region", "virginia"),
				),
			},
		},
	})
}

func testAccCheckHerokuSpace_basic(spaceName string, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}
`, spaceName, orgName)
}

func testAccCheckHerokuSpaceWithDatasource_basic(spaceName string, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}

data "heroku_space" "foobar" {
  name = "${heroku_space.foobar.name}"
}
`, spaceName, orgName)
}
