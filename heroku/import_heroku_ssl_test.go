package heroku

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccHerokuSSL_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	keyFile := wd + "/test-fixtures/terraform.key"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSSLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuSSLConfig(appName, certFile, keyFile),
			},
			{
				ResourceName:        "heroku_ssl.one",
				ImportStateIdPrefix: appName + ":",
				ImportState:         true,
				ImportStateVerify:   true,
			},
		},
	})
}
