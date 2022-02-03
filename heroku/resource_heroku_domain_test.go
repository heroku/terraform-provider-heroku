package heroku

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
	"github.com/heroku/terraform-provider-heroku/v4/helper/test"
)

func TestAccHerokuDomain_Basic(t *testing.T) {
	var domain heroku.Domain
	var endpoint heroku.SniEndpoint
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDomainConfig_basic(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDomainExists("heroku_domain.one", &domain),
					testAccCheckHerokuDomainAttributes(&domain, &endpoint),
					resource.TestCheckResourceAttr("heroku_domain.one", "hostname", "terraform-tftest-"+randString+".example.com"),
					resource.TestCheckResourceAttrSet("heroku_domain.one", "app_id"),
				),
			},
		},
	})
}

func TestAccHerokuDomain_No_SSL_Change(t *testing.T) {
	var domain heroku.Domain
	var endpoint heroku.SniEndpoint
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDomainConfig_ssl_no_association(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDomainExists("heroku_domain.one", &domain),
					testAccCheckHerokuSSLExists("heroku_ssl.one", &endpoint),
					testAccCheckHerokuDomainAttributes(&domain, &endpoint),
					resource.TestCheckNoResourceAttr("heroku_domain.one", "sni_endpoint_id"),
					resource.TestCheckResourceAttr("heroku_domain.one", "hostname", "terraform-tftest-"+randString+".example.com"),
					resource.TestCheckResourceAttrSet("heroku_domain.one", "app_id"),
				),
			},
			{
				PreConfig: test.Sleep(t, 15),
				Config:    testAccCheckHerokuDomainConfig_ssl(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDomainExists("heroku_domain.one", &domain),
					testAccCheckHerokuSSLExists("heroku_ssl.one", &endpoint),
					testAccCheckHerokuDomainAttributes(&domain, &endpoint),
					resource.TestCheckResourceAttrPtr("heroku_domain.one", "sni_endpoint_id", &endpoint.ID),
					resource.TestCheckResourceAttr("heroku_domain.one", "hostname", "terraform-tftest-"+randString+".example.com"),
					resource.TestCheckResourceAttrSet("heroku_domain.one", "app_id"),
				),
			},
		},
	})
}

func TestAccHerokuDomain_SSL(t *testing.T) {
	var domain heroku.Domain
	var endpoint heroku.SniEndpoint
	randString := acctest.RandString(10)
	appName := fmt.Sprintf("tftest-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuDomainConfig_ssl(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDomainExists("heroku_domain.one", &domain),
					testAccCheckHerokuSSLExists("heroku_ssl.one", &endpoint),
					testAccCheckHerokuDomainAttributes(&domain, &endpoint),
					resource.TestCheckResourceAttrPtr("heroku_domain.one", "sni_endpoint_id", &endpoint.ID),
					resource.TestCheckResourceAttr("heroku_domain.one", "hostname", "terraform-tftest-"+randString+".example.com"),
					resource.TestCheckResourceAttrSet("heroku_domain.one", "app_id"),
				),
			},
			{
				PreConfig: test.Sleep(t, 15),
				Config:    testAccCheckHerokuDomainConfig_ssl_change(appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuDomainExists("heroku_domain.one", &domain),
					testAccCheckHerokuSSLExists("heroku_ssl.two", &endpoint),
					testAccCheckHerokuDomainAttributes(&domain, &endpoint),
					resource.TestCheckResourceAttrPtr("heroku_domain.one", "sni_endpoint_id", &endpoint.ID),
					resource.TestCheckResourceAttr("heroku_domain.one", "hostname", "terraform-tftest-"+randString+".example.com"),
					resource.TestCheckResourceAttrSet("heroku_domain.one", "app_id"),
				),
			},
		},
	})
}

func testAccCheckHerokuDomainDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_domain" {
			continue
		}

		_, err := client.DomainInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Domain still exists")
		}
	}

	return nil
}

func testAccCheckHerokuDomainAttributes(Domain *heroku.Domain, endpoint *heroku.SniEndpoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.HasPrefix(Domain.Hostname, "terraform-") && !strings.HasSuffix(Domain.Hostname, ".example.com") {
			return fmt.Errorf("Bad hostname: %s", Domain.Hostname)
		}

		if !strings.Contains(*Domain.CName, ".herokudns.com") {
			return fmt.Errorf("Expected cname to be [*.herokudns.com] but got: [%s]", *Domain.CName)
		}

		if v := Domain.SniEndpoint; v != nil {
			if v.ID != endpoint.ID {
				return fmt.Errorf("Expected sni_endpoint_id to be: %s but got: [%s]", v.ID, endpoint.ID)
			}
		}

		return nil
	}
}

func testAccCheckHerokuDomainExists(n string, Domain *heroku.Domain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Domain ID is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundDomain, err := client.DomainInfo(context.TODO(), rs.Primary.Attributes["app_id"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundDomain.ID != rs.Primary.ID {
			return fmt.Errorf("Domain not found")
		}

		*Domain = *foundDomain

		return nil
	}
}

func testAccCheckHerokuDomainConfig_ssl_no_association(appName string) string {
	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	keyFile := wd + "/test-fixtures/terraform.key"

	return fmt.Sprintf(`resource "heroku_app" "one" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "one" {
    app_id = heroku_app.one.id
    file_path = "test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}

resource "heroku_app_release" "one" {
  app_id = heroku_app.one.id
  slug_id = "${heroku_slug.one.id}"
}

resource "heroku_formation" "web" {
  app_id = heroku_app.one.id
  type = "web"
  size = "hobby"
  quantity = 1
  # Wait until the build has completed before attempting to scale
  depends_on = [heroku_app_release.one]
}

resource "heroku_ssl" "one" {
  app_id = heroku_app.one.uuid
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
  # Wait until the process_tier changes to hobby before attempting to create a cert
  depends_on = [heroku_formation.web]
}

resource "heroku_domain" "one" {
  app_id = heroku_app.one.id
  hostname = "terraform-%s.example.com"
  # Wait until the certificate has been created before adding domains to avoid auto-association. Once auto-association has been sunset we no longer need to do this. See https://devcenter.heroku.com/changelog-items/1938.
  depends_on = [heroku_ssl.one]
}`, appName, certFile, keyFile, appName)
}

func testAccCheckHerokuDomainConfig_ssl_change(appName string) string {
	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	keyFile := wd + "/test-fixtures/terraform.key"

	return fmt.Sprintf(`resource "heroku_app" "one" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "one" {
    app_id = heroku_app.one.id
    file_path = "test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}

resource "heroku_app_release" "one" {
  app_id = heroku_app.one.id
  slug_id = "${heroku_slug.one.id}"
}

resource "heroku_formation" "web" {
  app_id = heroku_app.one.id
  type = "web"
  size = "hobby"
  quantity = 1
  depends_on = [heroku_app_release.one]
}

resource "heroku_ssl" "one" {
  app_id = heroku_app.one.uuid
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
  depends_on = [heroku_formation.web]
}

resource "heroku_ssl" "two" {
  app_id = heroku_app.one.uuid
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
  depends_on = [heroku_formation.web]
}

resource "heroku_domain" "one" {
  app_id = heroku_app.one.id
  hostname = "terraform-%s.example.com"
  sni_endpoint_id = "${heroku_ssl.two.id}"
}`, appName, certFile, keyFile, certFile, keyFile, appName)
}

func testAccCheckHerokuDomainConfig_ssl(appName string) string {
	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	keyFile := wd + "/test-fixtures/terraform.key"

	return fmt.Sprintf(`resource "heroku_app" "one" {
    name = "%s"
    region = "us"
}

resource "heroku_slug" "one" {
    app_id = heroku_app.one.id
    file_path = "test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}

resource "heroku_app_release" "one" {
  app_id = heroku_app.one.id
  slug_id = "${heroku_slug.one.id}"
}

resource "heroku_formation" "web" {
  app_id = heroku_app.one.id
  type = "web"
  size = "hobby"
  quantity = 1
  depends_on = [heroku_app_release.one]
}

resource "heroku_ssl" "one" {
  app_id = heroku_app.one.uuid
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
  depends_on = [heroku_formation.web]
}

resource "heroku_domain" "one" {
  app_id = heroku_app.one.id
  hostname = "terraform-%s.example.com"
  sni_endpoint_id = "${heroku_ssl.one.id}"
}`, appName, certFile, keyFile, appName)
}

func testAccCheckHerokuDomainConfig_basic(appName string) string {
	return fmt.Sprintf(`resource "heroku_app" "one" {
    name = "%s"
    region = "us"
}

resource "heroku_domain" "one" {
  app_id = heroku_app.one.id
  hostname = "terraform-%s.example.com"
}`, appName, appName)
}
