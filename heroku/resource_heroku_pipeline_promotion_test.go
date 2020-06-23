package heroku

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	heroku "github.com/heroku/heroku-go/v5"
)

func TestAccHerokuPipelinePromotionSingleTarget_Basic(t *testing.T) {
	var sourceapp, targetapp1, targetapp2 heroku.App
	var pipeline heroku.Pipeline
	var apprelease heroku.Release
	var pipelineCouplingSource, pipelineCouplingTarget1, pipelineCouplingTarget2 heroku.PipelineCoupling
	var promotion heroku.PipelinePromotion

	sourceApp := fmt.Sprintf("tftest-source-%s", acctest.RandString(10))
	targetApp1 := fmt.Sprintf("tftest-target1-%s", acctest.RandString(10))
	targetApp2 := fmt.Sprintf("tftest-target2-%s", acctest.RandString(10))
	pipelineName := fmt.Sprintf("tftest-pipeline-%s", acctest.RandString(10))
	pipelineOwnerID := testAccConfig.GetUserIDOrSkip(t)
	appReleaseSlugID := testAccConfig.GetSlugIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				PreConfig: sleep(t, 15),
				Config:    testAccCheckHerokuPipelinePromotionSingleTarget_basic(sourceApp, targetApp1, targetApp2, pipelineName, pipelineOwnerID, appReleaseSlugID),
				Check: resource.ComposeTestCheckFunc(
					// Setup a source app
					testAccCheckHerokuAppExists("heroku_app.foobar-source-app", &sourceapp),
					// Setup two target apps
					testAccCheckHerokuAppExists("heroku_app.foobar-target-app1", &targetapp1),
					testAccCheckHerokuAppExists("heroku_app.foobar-target-app2", &targetapp2),
					// Setupp a pipeline
					testAccCheckHerokuPipelineExists("heroku_pipeline.foobar-pipeline", &pipeline),
					// Setup an app release associated with the source app
					testAccCheckHerokuAppReleaseExists("heroku_app_release.foobar-release", &apprelease),
					// Setup pipeline couplings for all three apps to connect them to the pipeline
					testAccCheckHerokuPipelineCouplingExists("heroku_pipeline_coupling.foobar-source-coupling", &pipelineCouplingSource),
					testAccCheckHerokuPipelineCouplingExists("heroku_pipeline_coupling.foobar-target-coupling1", &pipelineCouplingTarget1),
					testAccCheckHerokuPipelineCouplingExists("heroku_pipeline_coupling.foobar-target-coupling2", &pipelineCouplingTarget2),
					// Setup the pipeline promotion
					testAccCheckHerokuPipelinePromotionExists("heroku_pipeline_promotion.foobar-promotion", &promotion),
					// Verify the promotion succeeded/completed
					resource.TestCheckResourceAttr("heroku_pipeline_promotion.foobar-promotion", "status", "completed"),
					// Verify promotion attributes
					resource.TestCheckResourceAttrPair("heroku_pipeline_promotion.foobar-promotion", "pipeline", "heroku_pipeline.foobar-pipeline", "id"),
					resource.TestCheckResourceAttrPair("heroku_pipeline_promotion.foobar-promotion", "source", "heroku_app.foobar-source-app", "uuid"),
					resource.TestCheckResourceAttrPair("heroku_pipeline_promotion.foobar-promotion", "release_id", "heroku_app_release.foobar-release", "id"),
					resource.TestCheckResourceAttrPair("heroku_pipeline_promotion.foobar-promotion", "targets.0", "heroku_app.foobar-target-app1", "uuid"),
					resource.TestCheckResourceAttrPair("heroku_pipeline_promotion.foobar-promotion", "targets.1", "heroku_app.foobar-target-app2", "uuid"),
				),
			},
		},
	})
}

func testAccCheckHerokuPipelinePromotionSingleTarget_basic(sourceApp, targetApp1, targetApp2, pipelineName, pipelineOwnerID, appReleaseSlugID string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar-source-app" {
	name = "%s"
	region = "us"
}
resource "heroku_app" "foobar-target-app1" {
	name = "%s"
	region = "us"
}
resource "heroku_app" "foobar-target-app2" {
	name = "%s"
	region = "us"
}
resource "heroku_pipeline" "foobar-pipeline" {
	name = "%s"
	owner {
		id = "%s"
		type = "user"
	}
}
resource "heroku_app_release" "foobar-release" {
	app = "${heroku_app.foobar-source-app.id}"
	slug_id = "%s"
}
resource "heroku_pipeline_coupling" "foobar-source-coupling" {
	app      = "${heroku_app.foobar-source-app.id}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "development"
}
resource "heroku_pipeline_coupling" "foobar-target-coupling1" {
	app      = "${heroku_app.foobar-target-app1.id}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "development"
}
resource "heroku_pipeline_coupling" "foobar-target-coupling2" {
	app      = "${heroku_app.foobar-target-app2.id}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "development"
}
resource "heroku_pipeline_promotion" "foobar-promotion" {
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	source = "${heroku_app.foobar-source-app.uuid}"
	targets = ["${heroku_app.foobar-target-app1.uuid}","${heroku_app.foobar-target-app2.uuid}"]
}
`, sourceApp, targetApp1, targetApp2, pipelineName, pipelineOwnerID, appReleaseSlugID)
}

func testAccCheckHerokuPipelinePromotionExists(n string, promotion *heroku.PipelinePromotion) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No pipeline promotion name set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundPipelinePromotion, err := client.PipelinePromotionInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundPipelinePromotion.ID != rs.Primary.ID {
			return fmt.Errorf("Pipeline promotion not found")
		}

		*promotion = *foundPipelinePromotion

		log.Printf("[DEBUG] PipelinePromotion found: %q\n", *promotion)

		return nil
	}
}
