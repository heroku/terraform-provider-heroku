package heroku

import (
  "context"
  "fmt"
  "io/ioutil"
  "os"
  "strings"
  "testing"

  "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
  "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
  "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
  heroku "github.com/heroku/heroku-go/v5"
)

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

  slugID := testAccConfig.GetSlugIDOrSkip(t)

  resource.Test(t, resource.TestCase{
    PreCheck:     func() { testAccPreCheck(t) },
    Providers:    testAccProviders,
    CheckDestroy: testAccCheckHerokuCertDestroy,
    Steps: []resource.TestStep{
      {
        Config: testAccCheckHerokuCertConfig(appName, slugID, certFile2, keyFile2),
        Check: resource.ComposeTestCheckFunc(
          testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
          testAccCheckHerokuCertificateChain(&endpoint, certificateChain2),
        ),
      },
      {
        Config: testAccCheckHerokuCertConfig(appName, slugID, certFile, keyFile),
        Check: resource.ComposeTestCheckFunc(
          testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
          testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
        ),
      },
    },
  })
}

func testAccCheckHerokuCertConfig(appName, slugID, certFile, keyFile string) string {
  return strings.TrimSpace(fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name = "%s"
  region = "us"
}

# Unfortunately the only way to set the process tier to one compatible with
# Heroku SSL is to scale the app, and we can only scale it if there is a
# release.
resource "heroku_app_release" "foobar-release" {
  app = "${heroku_app.foobar.name}"
  slug_id = "%s"
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
}`, appName, slugID, certFile, keyFile))
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
