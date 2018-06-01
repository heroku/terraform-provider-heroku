package heroku

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDatasourceHerokuSpacePeeringInfo_Basic(t *testing.T) {
	spaceName := fmt.Sprintf("tftest-space-peer-info-%s", acctest.RandString(3))
	orgName := os.Getenv("HEROKU_SPACES_ORGANIZATION")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if orgName == "" {
				t.Skip("HEROKU_SPACES_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSpacePeeringInfo_basic(spaceName, orgName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_space.foobar", "name", spaceName),
					resource.TestCheckResourceAttrSet(
						"data.heroku_space_peering_info.foobar", "aws_account_id"),
					resource.TestCheckResourceAttr(
						"data.heroku_space_peering_info.foobar", "aws_region", "us-east-1"),
					resource.TestCheckResourceAttrSet(
						"data.heroku_space_peering_info.foobar", "vpc_id"),
				),
			},
		},
	})
}

func testAccCheckHerokuSpacePeeringInfo_basic(spaceName string, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}

data "heroku_space_peering_info" "foobar" {
  name = "${heroku_space.foobar.name}"
}
`, spaceName, orgName)
}
