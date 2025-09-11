package heroku

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v6"
)

// Generates a "test step" not a whole test, so that it can reuse the space.
// See: resource_heroku_space_test.go, where this is used.
func testStep_AccHerokuVPNConnection_Basic(t *testing.T, spaceConfig string) resource.TestStep {
	var vpnConnection heroku.VPNConnection
	return resource.TestStep{
		ResourceName: "heroku_space_vpn_connection.foobar",
		Config:       testAccCheckHerokuVPNConnectionConfig_basic(spaceConfig),
		Check: resource.ComposeTestCheckFunc(
			testAccCheckHerokuVPNConnectionExists("heroku_space_vpn_connection.foobar", &vpnConnection),
			testAccCheckHerokuVPNConnectionAttributes(&vpnConnection),
			resource.TestCheckResourceAttr(
				"heroku_space_vpn_connection.foobar", "space_cidr_block", "10.0.0.0/16"),
			resource.TestCheckResourceAttr(
				"heroku_space_vpn_connection.foobar", "ike_version", "1"),
			resource.TestCheckResourceAttr(
				"heroku_space_vpn_connection.foobar", "tunnels.#", "2"),
		),
	}
}

func testAccCheckHerokuVPNConnectionExists(n string, vpnConnection *heroku.VPNConnection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VPN connection ID set")
		}
		space, id, _ := parseCompositeID(rs.Primary.ID)

		client := testAccProvider.Meta().(*Config).Api
		foundVPNConnection, err := client.VPNConnectionInfo(context.TODO(), space, id)
		if err != nil {
			return err
		}

		if foundVPNConnection.ID != id {
			return fmt.Errorf("VPN connection not found")
		}

		*vpnConnection = *foundVPNConnection

		return nil
	}
}

func testAccCheckHerokuVPNConnectionAttributes(vpnConnection *heroku.VPNConnection) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vpnConnection.Name != "foobar" {
			return fmt.Errorf("Bad VPN connection name: got %v, want %v", vpnConnection.Name, "foobar")
		}
		if !reflect.DeepEqual(vpnConnection.RoutableCidrs, []string{"10.100.0.0/16"}) {
			return fmt.Errorf("Bad VPN routable CIDRs: got %v, want %v", vpnConnection.RoutableCidrs, []string{"10.100.0.0/16"})
		}

		return nil
	}
}

func testAccCheckHerokuVPNConnectionConfig_basic(spaceConfig string) string {
	return fmt.Sprintf(`
# heroku_space.foobar config inherited from previous steps
%s

resource "heroku_space_vpn_connection" "foobar" {
	space          = heroku_space.foobar.name
	name           = "foobar"
	public_ip      = element(heroku_space.foobar.outbound_ips, 0)
	routable_cidrs = ["10.100.0.0/16"]
}
`, spaceConfig)
}

func TestHerokuSpaceVPNConnectionGeneration(t *testing.T) {
	tests := []struct {
		name        string
		generation  string
		expectError bool
		description string
	}{
		{
			name:        "Cedar generation should be supported",
			generation:  "cedar",
			expectError: false,
			description: "Cedar supports VPN connections",
		},
		{
			name:        "Fir generation should be unsupported",
			generation:  "fir",
			expectError: true,
			description: "Fir does not support VPN connections",
		},
		{
			name:        "Default generation (cedar) should be supported",
			generation:  "", // Will default to cedar
			expectError: false,
			description: "Default cedar generation supports VPN connections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the feature support logic
			generation := tt.generation
			if generation == "" {
				generation = "cedar" // Default
			}

			supported := IsFeatureSupported(generation, "space", "vpn_connection")
			shouldError := !supported

			if shouldError != tt.expectError {
				t.Errorf("Expected error: %t, but got: %t for generation %s",
					tt.expectError, shouldError, generation)
			}

			t.Logf("✅ Generation: %s, Supported: %t, ShouldError: %t",
				generation, supported, shouldError)
		})
	}
}
