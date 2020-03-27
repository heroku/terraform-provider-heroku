package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"regexp"
	"testing"
)

func TestAccHerokuPipelineConfigVar_importBasic(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	stage := "test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipelineConfigVar_basic(name, stage),
			},
			{
				ResourceName:      "heroku_pipeline_config_var.configs",
				ImportState:       true,
				ImportStateVerify: true,
				ExpectError:       regexp.MustCompile(`not possible to import`),
			},
		},
	})
}
