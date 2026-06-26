package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHerokuPipeline_importBasic(t *testing.T) {
	pName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	ownerID := testAccConfig.GetUserIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHerokuPipelineDestroy,
		Steps: []resource.TestStep{
			{
				// Use a config with an explicit owner block: owner is now an
				// optional (non-Computed) block, so it is only tracked in state
				// when configured. Import reflects the remote owner, so the
				// baseline must include it for ImportStateVerify to match.
				Config: testAccCheckHerokuPipeline_basic(pName, ownerID, "user"),
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
