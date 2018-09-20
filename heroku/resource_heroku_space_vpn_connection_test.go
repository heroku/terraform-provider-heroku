package heroku

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/heroku-go/v3"
)

func TestAccHerokuVPNConnection_basic(t *testing.T) {
	var vpnConnection heroku.VPNConnection
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := testAccConfig.GetSpaceOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuVPNConnectionDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space_vpn_connection.foobar",
				Config:       testAccCheckHerokuVPNConnectionConfig_basic(spaceName, org),
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
			},
		},
	})
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
		space, id := parseCompositeID(rs.Primary.ID)

		config := testAccProvider.Meta().(*Config)
		foundVPNConnection, err := config.Api.VPNConnectionInfo(context.TODO(), space, id)
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

func testAccCheckHerokuVPNConnectionDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_space_vpn_connection" {
			continue
		}

		space, id := parseCompositeID(rs.Primary.ID)
		_, err := config.Api.VPNConnectionInfo(context.TODO(), space, id)
		if err == nil {
			return fmt.Errorf("VPN connection still exists")
		}
	}

	return nil
}

func testAccCheckHerokuVPNConnectionConfig_basic(spaceName string, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}

resource "heroku_space_vpn_connection" "foobar" {
	space          = "${heroku_space.foobar.name}"
	name           = "foobar"
	public_ip      = "${element(heroku_space.foobar.outbound_ips, 0)}"
	routable_cidrs = ["10.100.0.0/16"]
}
`, spaceName, orgName)
}
