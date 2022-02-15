package heroku

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
	"github.com/heroku/terraform-provider-heroku/v4/helper/test"
)

// We break apart testing for EU and US because at present, Heroku deals with
// each a bit differently and the setup/teardown of separate tests seems to
// help them to perform more consistently.
// https://devcenter.heroku.com/articles/ssl-endpoint#add-certificate-and-intermediaries
//
// We also have a time.Sleep() set for the update step (step 2 of 2) in each
// region's tests. This is somewhat kludgy, but the Heroku API SSL Endpoint
// handles parts of the create and update requests asynchronously, and if you
// add a cert+key then immediately update it, and then delete it (end of test),
// there are scenarios where the operations get out of order. For now, sleeping
// on update seems to allow the test to run smoothly; in real life, this test
// case is definitely an extreme edge case.
func TestAccHerokuCert_EU(t *testing.T) {
	t.Skip("SSL Endpoint shutdown: https://devcenter.heroku.com/changelog-items/2280")

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
		CheckDestroy: testAccCheckHerokuCertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCertEUConfig(appName, certFile, keyFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
					resource.TestMatchResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", regexp.MustCompile(`^[^\.]+.ssl.herokudns.com$`)),
				),
			},
			{
				PreConfig: test.Sleep(t, 15),
				Config:    testAccCheckHerokuCertEUConfig(appName, certFile2, keyFile2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain2),
					resource.TestMatchResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", regexp.MustCompile(`^[^\.]+.ssl.herokudns.com$`)),
				),
			},
		},
	})
}

func TestAccHerokuCert_US(t *testing.T) {
	t.Skip("SSL Endpoint shutdown: https://devcenter.heroku.com/changelog-items/2280")

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
		CheckDestroy: testAccCheckHerokuCertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCertUSConfig(appName, certFile2, keyFile2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain2),
					resource.TestMatchResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", regexp.MustCompile(`^[^\.]+.ssl.herokudns.com$`)),
				),
			},
			{
				PreConfig: test.Sleep(t, 15),
				Config:    testAccCheckHerokuCertUSConfig(appName, certFile, keyFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
					resource.TestMatchResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", regexp.MustCompile(`^[^\.]+.ssl.herokudns.com$`)),
				),
			},
		},
	})
}

func testAccCheckHerokuCertEUConfig(appName, certFile, keyFile string) string {
	return strings.TrimSpace(fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name = "%s"
  region = "eu"
}

resource "heroku_addon" "ssl" {
  app_id = heroku_app.foobar.id
  plan = "ssl:endpoint"
}

resource "heroku_cert" "ssl_certificate" {
  app = "${heroku_app.foobar.name}"
  depends_on = ["heroku_addon.ssl"]
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
}`, appName, certFile, keyFile))
}

func testAccCheckHerokuCertUSConfig(appName, certFile, keyFile string) string {
	return strings.TrimSpace(fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name = "%s"
  region = "us"
}

resource "heroku_addon" "ssl" {
  app_id = heroku_app.foobar.id
  plan = "ssl:endpoint"
}

resource "heroku_cert" "ssl_certificate" {
  app = "${heroku_app.foobar.name}"
  depends_on = ["heroku_addon.ssl"]
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
}`, appName, certFile, keyFile))
}

func testAccCheckHerokuCertDestroy(s *terraform.State) error {
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

func testAccCheckHerokuCertificateChain(endpoint *heroku.SSLEndpoint, chain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if endpoint.CertificateChain != chain {
			return fmt.Errorf("Bad certificate chain: %s", endpoint.CertificateChain)
		}

		return nil
	}
}

func testAccCheckHerokuCertExists(n string, endpoint *heroku.SSLEndpoint) resource.TestCheckFunc {
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
