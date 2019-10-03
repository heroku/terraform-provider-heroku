package heroku

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccHerokuCert_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	keyFile := wd + "/test-fixtures/terraform.key"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuCertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCertUSConfig(appName, certFile, keyFile),
			},
			{
				ResourceName:        "heroku_cert.ssl_certificate",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
			},
		},
	})
}
