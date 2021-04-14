package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"testing"
)

func TestAccHerokuReviewAppConfig_importBasic(t *testing.T) {
	pipelineID := testAccConfig.GetPipelineIDorSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuReviewAppConfig_basic(pipelineID, "true",
					"true", "true"),
			},
			{
				ResourceName:      "heroku_review_app_config.foobar",
				ImportStateIdFunc: testAccHerokuReviewAppConfigImportStateIDFunc("heroku_review_app_config.foobar"),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccHerokuReviewAppConfigImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("[ERROR] Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s:%s", rs.Primary.Attributes["pipeline_id"], rs.Primary.Attributes["org_repo"]), nil
	}
}
