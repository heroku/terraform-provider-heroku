package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHerokuPipeline_importBasic(t *testing.T) {
	pName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHerokuPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipeline_NoOwner(pName),
			},
			{
				ResourceName:            "heroku_pipeline.foobar",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_vars"},
			},
		},
	})
}
