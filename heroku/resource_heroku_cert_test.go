package heroku

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuCert_Legacy(t *testing.T) {
	var endpoint heroku.SSLEndpoint
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	certFile2 := wd + "/test-fixtures/terraform2.cert"
	keyFile := wd + "/test-fixtures/terraform.key"
	keyFile2 := wd + "/test-fixtures/terraform2.key"

	certificateChainBytes, _ := ioutil.ReadFile(certFile)
	certificateChain := string(certificateChainBytes)
	certificateChain2Bytes, _ := ioutil.ReadFile(certFile2)
	certificateChain2 := string(certificateChain2Bytes)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuCertDestroyLegacy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCertLegacyConfig(appName, certFile2, keyFile2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuLegacyCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuLegacyCertificateChain(&endpoint, certificateChain2),
					resource.TestMatchResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", regexp.MustCompile(`^[^\.]+.ssl.herokudns.com$`)),
				),
			},
			{
				PreConfig: sleep(t, 15),
				Config:    testAccCheckHerokuCertLegacyConfig(appName, certFile, keyFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuLegacyCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuLegacyCertificateChain(&endpoint, certificateChain),
					resource.TestMatchResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", regexp.MustCompile(`^[^\.]+.ssl.herokudns.com$`)),
				),
			},
		},
	})
}

func TestAccHerokuCert(t *testing.T) {
	var endpoint heroku.SniEndpoint
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	certFile2 := wd + "/test-fixtures/terraform2.cert"
	keyFile := wd + "/test-fixtures/terraform.key"
	keyFile2 := wd + "/test-fixtures/terraform2.key"

	certificateChainBytes, _ := ioutil.ReadFile(certFile)
	certificateChain := string(certificateChainBytes)
	certificateChain2Bytes, _ := ioutil.ReadFile(certFile2)
	certificateChain2 := string(certificateChain2Bytes)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuCertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCertConfig(appName, certFile2, keyFile2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain2),
				),
			},
			{
				PreConfig: sleep(t, 15),
				Config:    testAccCheckHerokuCertConfig(appName, certFile, keyFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
				),
			},
		},
	})
}

func testAccCheckHerokuCertConfig(appName, certFile, keyFile string) string {
	return strings.TrimSpace(fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name = "%s"
  region = "us"
}

resource "heroku_slug" "foobar" {
    app = "${heroku_app.foobar.name}"
    file_path = "test-fixtures/slug.tgz"
    process_types = {
      web = "ruby server.rb"
    }
}

resource "heroku_app_release" "foobar-release" {
  app = "${heroku_app.foobar.name}"
  slug_id = "${heroku_slug.foobar.id}"
}

resource "heroku_formation" "foobar-web" {
  app = "${heroku_app.foobar.name}"
  type = "web"
  size = "hobby"
  quantity = 1
  depends_on = ["heroku_app_release.foobar-release"]
}

resource "heroku_cert" "ssl_certificate" {
  app = "${heroku_app.foobar.name}"
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
  depends_on = ["heroku_formation.foobar-web"]
}`, appName, certFile, keyFile))
}

func testAccCheckHerokuCertDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_cert" {
			continue
		}

		_, err := client.SniEndpointInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Cerfificate still exists")
		}
	}

	return nil
}

func sleep(t *testing.T, amount time.Duration) func() {
	return func() {
		time.Sleep(amount * time.Second)
	}
}

func testAccCheckHerokuCertLegacyConfig(appName, certFile, keyFile string) string {
	return strings.TrimSpace(fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name = "%s"
  region = "us"
}
resource "heroku_addon" "ssl" {
  app = "${heroku_app.foobar.name}"
  plan = "ssl:endpoint"
}
resource "heroku_cert" "ssl_certificate" {
  app = "${heroku_app.foobar.name}"
  depends_on = ["heroku_addon.ssl"]
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
}`, appName, certFile, keyFile))
}

func createSslEndpoint(app, cert, key string) error {
	client := testAccProvider.Meta().(*Config).Api

	preprocess := true
	opts := heroku.SSLEndpointCreateOpts{
		CertificateChain: cert,
		Preprocess:       &preprocess,
		PrivateKey:       key,
	}
	_, err := client.SSLEndpointCreate(context.TODO(), app, opts)

	if err == nil {
		return fmt.Errorf("Cerfificate still exists")
	}

	return nil
}

func testAccCheckHerokuCertDestroyLegacy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_cert" {
			continue
		}

		_, err := client.SSLEndpointInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Cerfificate still exists")
		}
	}

	return nil
}

func testAccCheckHerokuCertificateChain(endpoint *heroku.SniEndpoint, chain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if endpoint.CertificateChain != chain {
			return fmt.Errorf("Bad certificate chain: %s", endpoint.CertificateChain)
		}

		return nil
	}
}

func testAccCheckHerokuCertExists(n string, endpoint *heroku.SniEndpoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNI endpoint ID is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundEndpoint, err := client.SniEndpointInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundEndpoint.ID != rs.Primary.ID {
			return fmt.Errorf("SNI endpoint not found")
		}

		*endpoint = *foundEndpoint

		return nil
	}
}

func testAccCheckHerokuLegacyCertificateChain(endpoint *heroku.SSLEndpoint, chain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if endpoint.CertificateChain != chain {
			return fmt.Errorf("Bad certificate chain: %s", endpoint.CertificateChain)
		}

		return nil
	}
}

func testAccCheckHerokuLegacyCertExists(n string, endpoint *heroku.SSLEndpoint) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSL endpoint ID is set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundEndpoint, err := client.SSLEndpointInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundEndpoint.ID != rs.Primary.ID {
			return fmt.Errorf("SSL endpoint not found")
		}

		*endpoint = *foundEndpoint

		return nil
	}
}
