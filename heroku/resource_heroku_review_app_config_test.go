package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccHerokuReviewAppConfig_Basic(t *testing.T) {
	pipelineID := testAccConfig.GetPipelineIDorSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuReviewAppConfig_basic(pipelineID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"heroku_review_app_config.foobar", "pipeline_id"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "org_repo", "heroku/ruby-getting-started"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "automatic_review_apps", "false"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "base_name", "ruby-st"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "deploy_target.0.id", "us"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "deploy_target.0.type", "region"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "destroy_stale_apps", "true"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "stale_days", "5"),
					resource.TestCheckResourceAttr(
						"heroku_review_app_config.foobar", "wait_for_ci", "false"),
					resource.TestCheckResourceAttrSet(
						"heroku_review_app_config.foobar", "repo_id"),
				),
			},
		},
	})
}

func testAccCheckHerokuReviewAppConfig_basic(pipelineID string) string {
	return fmt.Sprintf(`
data "heroku_pipeline" "foobar" {
  name = "%s"
}

resource "heroku_review_app_config" "foobar" {
  pipeline_id = data.heroku_pipeline.foobar.id
  org_repo = "heroku/ruby-getting-started"
  automatic_review_apps = false
  base_name = "ruby-st"

  deploy_target {
    id = "us"
    type = "region"
  }

  destroy_stale_apps = true
  stale_days = 5
  wait_for_ci = false
}
`, pipelineID)
}
