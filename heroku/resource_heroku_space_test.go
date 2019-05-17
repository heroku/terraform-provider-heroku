package heroku

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuSpace_Basic(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	spaceName2 := fmt.Sprintf("tftest2-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space.foobar",
				Config:       testAccCheckHerokuSpaceConfig_basic(spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "trusted_ip_ranges.#", "2"),
					testAccCheckHerokuSpaceAttributes(&space, spaceName),
					resource.TestCheckResourceAttrSet(
						"heroku_space.foobar", "outbound_ips.#"),
				),
			},
			{
				ResourceName: "heroku_space.foobar",
				Config:       testAccCheckHerokuSpaceConfig_basic(spaceName2, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					testAccCheckHerokuSpaceAttributes(&space, spaceName2),
				),
			},
		},
	})
}

func TestAccHerokuSpace_Shield(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tfshieldtest-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSpaceConfig_shield(spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					testAccCheckHerokuSpaceAttributes(&space, spaceName),
					resource.TestCheckResourceAttr(
						"heroku_space.foobar", "shield", "true"),
				),
			},
		},
	})
}

func TestAccHerokuSpace_IPRange(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space.foobar",
				Config:       testAccCheckHerokuSpaceConfig_iprange(spaceName, org, []string{"8.8.8.8/32"}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "trusted_ip_ranges.#", "1"),
				),
			},
			{
				ResourceName: "heroku_space.foobar",
				Config:       testAccCheckHerokuSpaceConfig_iprange(spaceName, org, []string{"8.8.8.8/32", "8.8.8.0/24"}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "trusted_ip_ranges.#", "2"),
				),
			},
		},
	})
}

func TestAccHerokuSpace_CIDRs(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tfcidrtest-%s", acctest.RandString(10))
	org := testAccConfig.GetAnyOrganizationOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space.foobar",
				Config:       testAccCheckHerokuSpaceConfig_cidr(spaceName, org, "10.0.0.0/16"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "cidr", "10.0.0.0/16"),
				),
			},
			{
				ResourceName: "heroku_space.foobar",
				Config:       testAccCheckHerokuSpaceConfig_data_cidr(spaceName, org, "10.1.0.0/20"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr("heroku_space.foobar", "data_cidr", "10.1.0.0/20"),
				),
			},
		},
	})
}

func testAccCheckHerokuSpaceConfig_basic(spaceName, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name = "%s"
  organization = "%s"
  region = "virginia"
  trusted_ip_ranges = [
    "8.8.8.8/32",
    "8.8.8.0/24",
  ]
}
`, spaceName, orgName)
}

func testAccCheckHerokuSpaceConfig_shield(spaceName, orgName string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
  shield       = true
}
`, spaceName, orgName)
}

func testAccCheckHerokuSpaceConfig_iprange(spaceName, orgName string, ips []string) string {
	ipsStr := fmt.Sprintf("\"%s\"", strings.Join(ips, "\", \""))
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
  trusted_ip_ranges = [%s]
}
`, spaceName, orgName, ipsStr)
}

func testAccCheckHerokuSpaceConfig_cidr(spaceName, orgName string, cidr string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  cidr         = "%s"
  region       = "virginia"
}
`, spaceName, orgName, cidr)
}

func testAccCheckHerokuSpaceConfig_data_cidr(spaceName, orgName string, dataCidr string) string {
	return fmt.Sprintf(`
resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  data_cidr    = "%s"
  region       = "virginia"
}
`, spaceName, orgName, dataCidr)
}

func testAccCheckHerokuSpaceExists(n string, space *heroku.Space) resource.TestCheckFunc {
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

		if foundSpace.ID != rs.Primary.ID {
			return fmt.Errorf("Space not found")
		}

		*space = *foundSpace

		return nil
	}
}

func testAccCheckHerokuSpaceAttributes(space *heroku.Space, spaceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if space.Name != spaceName {
			return fmt.Errorf("Bad name: %s", space.Name)
		}

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
