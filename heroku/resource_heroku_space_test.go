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
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "log_drain_url"),
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

func TestAccHerokuSpaceLogDrain(t *testing.T) {
	var space spaceWithNAT
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	spaceConfigWithLogDrain1 := testAccCheckHerokuShieldSpaceConfig_withLogDrain(spaceName, org, "10.0.0.0/16", "https://drain1.example.com")
	spaceConfigWithLogDrain2 := testAccCheckHerokuShieldSpaceConfig_withLogDrain(spaceName, org, "10.0.0.0/16", "https://drain2.example.com")
	spaceConfigWithoutLogDrain := testAccCheckHerokuShieldSpaceConfig_withoutLogDrain(spaceName, org, "10.0.0.0/16")

	var space1Id string
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space.foobar",
				Config:       spaceConfigWithLogDrain1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "outbound_ips.#"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "data_cidr"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "log_drain_url", "https://drain1.example.com"),
					func(s *terraform.State) error {
						space1Id = space.ID
						return nil
					},
				),
			},
			{
				ResourceName: "heroku_space.foobar",
				Config:       spaceConfigWithLogDrain2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "outbound_ips.#"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "data_cidr"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "log_drain_url", "https://drain2.example.com"),
					func(s *terraform.State) error {
						if space1Id == "" {
							return fmt.Errorf("Space ID not set for space 1")
						}
						if space.ID != space1Id {
							return fmt.Errorf("Space ID changed: %s -> %s", space1Id, space.ID)
						}
						return nil
					},
				),
			},
			{
				ResourceName: "heroku_space.foobar",
				Config:       spaceConfigWithoutLogDrain,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "outbound_ips.#"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet("heroku_space.foobar", "data_cidr"),
					resource.TestCheckNoResourceAttr("heroku_space.foobar", "log_drain_url"),
					func(s *terraform.State) error {
						if space1Id == "" {
							return fmt.Errorf("Space ID not set for space 1")
						}
						if space.ID == space1Id {
							return fmt.Errorf("expected space id to change, but got same value: %s", space.ID)
						}
						return nil
					},
				),
			},
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
  cidr    	    = "%s"
  shield 		= true
}
`, spaceName, orgName, cidr)
}

func testAccCheckHerokuShieldSpaceConfig_withLogDrain(spaceName, orgName, cidr, logDrainURL string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
  region = "virginia"
  cidr    	    = "%s"
  shield 		= true
  log_drain_url = "%s"
}
`, spaceName, orgName, cidr, logDrainURL)
}

func testAccCheckHerokuShieldSpaceConfig_withoutLogDrain(spaceName, orgName, cidr string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
  region = "virginia"
  cidr    	    = "%s"
  shield 		= true
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
