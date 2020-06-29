package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccDatasourceHerokuPipeline_Basic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	pipelineName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipelineWithDatasourceBasic(appName, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.heroku_pipeline.foobar", "name", pipelineName),
					resource.TestCheckResourceAttrSet(
						"data.heroku_pipeline.foobar", "owner_id"),
					resource.TestCheckResourceAttrSet(
						"data.heroku_pipeline.foobar", "owner_type"),
					resource.TestCheckResourceAttrSet(
						"data.heroku_pipeline.foobar", "id"),
				),
			},
		},
	})
}

func testAccCheckHerokuPipelineWithDatasourceBasic(appName, pipelineName string) string {
	return fmt.Sprintf(`
resource "heroku_app" "staging" {
  name = "%s"
  region = "us"
}

resource "heroku_pipeline" "foobar" {
  name = "%s"
}

resource "heroku_pipeline_coupling" "staging" {
  app      = "${heroku_app.staging.id}"
  pipeline = "${heroku_pipeline.foobar.id}"
  stage    = "staging"
}

data "heroku_pipeline" "foobar" {
  name = "${heroku_pipeline_coupling.staging.pipeline}"
}
`, appName, pipelineName)
}
