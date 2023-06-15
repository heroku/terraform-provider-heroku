package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccHerokuSpace(t *testing.T) {
	var space spaceWithNAT
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)
	spaceConfig := testAccCheckHerokuSpaceConfig_basic(spaceName, org, "10.0.0.0/16")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space.foobar",
				Config:       spaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "outbound_ips.#"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "data_cidr"),
				),
			},
			// append space test Steps, sharing the space, instead of recreating for each test
			testStep_AccDatasourceHerokuSpace_Basic(t, spaceConfig),
			testStep_AccDatasourceHerokuSpacePeeringInfo_Basic(t, spaceConfig),
			testStep_AccHerokuApp_Space(t, spaceConfig, spaceName),
			testStep_AccHerokuApp_Space_Internal(t, spaceConfig, spaceName),
			testStep_AccHerokuSlug_WithFile_InPrivateSpace(t, spaceConfig),
			testStep_AccHerokuSpaceAppAccess_Basic(t, spaceConfig),
			testStep_AccHerokuSpaceAppAccess_importBasic(t, spaceName),
			testStep_AccHerokuSpaceInboundRuleset_Basic(t, spaceConfig),
			testStep_AccHerokuVPNConnection_Basic(t, spaceConfig),
		},
	})
}

// Permanently skipping Space_Shield test, as this is little more than an attribute test that takes at least 8-minutes to run.
// It's really just testing Shield space provisioning, which this Terraform provider is not responsible for validating.
//
// func TestAccHerokuSpace_Shield(t *testing.T) {
//  …
// }

func testAccCheckHerokuSpaceConfig_basic(spaceName, orgName, cidr string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
  region = "virginia"
  cidr         = "%s"
}
`, spaceName, orgName, cidr)
}

func testAccCheckHerokuSpaceExists(n string, space *spaceWithNAT) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No space name set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundSpace, err := client.SpaceInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		foundSpaceWithNAT := spaceWithNAT{
			Space: *foundSpace,
		}

		nat, err := client.SpaceNATInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}
		foundSpaceWithNAT.NAT = *nat

		if foundSpace.ID != rs.Primary.ID {
			return fmt.Errorf("Space not found")
		}

		*space = foundSpaceWithNAT

		return nil
	}
}

func testAccCheckHerokuSpaceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_space" {
			continue
		}

		_, err := client.SpaceInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Space still exists")
		}
	}

	return nil
}
