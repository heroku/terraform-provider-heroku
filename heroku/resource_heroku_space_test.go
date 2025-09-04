package heroku

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func TestAccHerokuSpace_Generation(t *testing.T) {
	var space spaceWithNAT
	spaceName := fmt.Sprintf("tftest-gen-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				// Test 1: Default generation (should be cedar)
				Config: testAccCheckHerokuSpaceConfig_generation(spaceName, org, "", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "generation", "cedar"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "shield", "false"),
				),
			},
			{
				// Test 2: Explicit cedar generation
				Config: testAccCheckHerokuSpaceConfig_generation(spaceName+"-cedar", org, "cedar", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "generation", "cedar"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "shield", "false"),
				),
			},
			{
				// Test 3: Fir generation (non-shield)
				Config: testAccCheckHerokuSpaceConfig_generation(spaceName+"-fir", org, "fir", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "generation", "fir"),
					resource.TestCheckResourceAttr("heroku_space.foobar", "shield", "false"),
				),
			},
		},
	})
}

func TestAccHerokuSpace_GenerationShieldValidation(t *testing.T) {
	spaceName := fmt.Sprintf("tftest-shield-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Test: Fir + Shield should fail during apply
				Config:      testAccCheckHerokuSpaceConfig_generation(spaceName, org, "fir", true),
				ExpectError: regexp.MustCompile("shield spaces are not supported for fir generation"),
			},
		},
	})
}

func TestAccHerokuSpace_GenerationForceNew(t *testing.T) {
	spaceName := fmt.Sprintf("tftest-forcenew-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create space with cedar generation
				Config: testAccCheckHerokuSpaceConfig_generation(spaceName, org, "cedar", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("heroku_space.foobar", "generation", "cedar"),
				),
			},
			{
				// Step 2: Change generation to fir - should force recreation
				Config: testAccCheckHerokuSpaceConfig_generation(spaceName, org, "fir", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("heroku_space.foobar", "generation", "fir"),
				),
			},
		},
	})
}

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

func testAccCheckHerokuSpaceConfig_generation(spaceName, orgName, generation string, shield bool) string {
	config := fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
  region = "virginia"
  cidr = "10.0.0.0/16"`, spaceName, orgName)

	if generation != "" {
		config += fmt.Sprintf(`
  generation = "%s"`, generation)
	}

	config += fmt.Sprintf(`
  shield = %t
}
`, shield)

	return config
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

// Unit tests for generation functionality
func TestHerokuSpaceGeneration(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "Resource created without generation defaults to cedar",
			config: map[string]interface{}{
				"name":         "test-space",
				"organization": "test-org",
			},
			expectError: false,
		},
		{
			name: "Cedar generation with non-shield space should succeed",
			config: map[string]interface{}{
				"name":         "test-space",
				"organization": "test-org",
				"generation":   "cedar",
				"shield":       false,
			},
			expectError: false,
		},
		{
			name: "Fir generation with non-shield space should succeed",
			config: map[string]interface{}{
				"name":         "test-space",
				"organization": "test-org",
				"generation":   "fir",
				"shield":       false,
			},
			expectError: false,
		},
		{
			name: "Fir generation with shield space should fail",
			config: map[string]interface{}{
				"name":         "test-space",
				"organization": "test-org",
				"generation":   "fir",
				"shield":       true,
			},
			expectError: true,
			errorMsg:    "shield spaces are not supported for fir generation",
		},
		{
			name: "Cedar generation with shield space should succeed",
			config: map[string]interface{}{
				"name":         "test-space",
				"organization": "test-org",
				"generation":   "cedar",
				"shield":       true,
			},
			expectError: false,
		},
		{
			name: "Default generation (cedar) with shield space should succeed",
			config: map[string]interface{}{
				"name":         "test-space",
				"organization": "test-org",
				"shield":       true,
				// generation not specified, should default to cedar
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create resource data from schema
			d := schema.TestResourceDataRaw(t, resourceHerokuSpace().Schema, tt.config)

			// Check default generation behavior
			generation := d.Get("generation").(string)
			if tt.config["generation"] == nil {
				if generation != "cedar" {
					t.Errorf("Expected default generation to be 'cedar', got '%s'", generation)
				}
			}

			// Test shield validation logic without actually calling the API
			shield := d.Get("shield").(bool)
			if shield {
				supported := IsFeatureSupported(generation, "space", "shield")
				if tt.expectError && supported {
					t.Errorf("Expected shield to be unsupported for %s generation", generation)
				}
				if !tt.expectError && !supported {
					t.Errorf("Expected shield to be supported for %s generation", generation)
				}
			}

			t.Logf("✅ Generation: %s, Shield: %t, Supported: %t", generation, shield, IsFeatureSupported(generation, "space", "shield"))
		})
	}
}
