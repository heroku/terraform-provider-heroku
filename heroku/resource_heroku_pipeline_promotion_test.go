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
	var pipeline heroku.Pipeline
	var release heroku.Release
	var pipelineCouplingSource heroku.PipelineCoupling
	var pipelineCouplingTarget heroku.PipelineCoupling
	var promotion heroku.PipelinePromotion

	sourceApp := fmt.Sprintf("tftest-source-%s", acctest.RandString(10))
	targetApp := fmt.Sprintf("tftest-target-%s", acctest.RandString(10))
	pipelineName := fmt.Sprintf("tftest-pipeline-%s", acctest.RandString(10))
	pipelineOwnerID := testAccConfig.GetUserIDOrSkip(t)
	appReleaseSlugID := testAccConfig.GetSlugIDOrSkip(t)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		// CheckDestroy: testAccCheckHerokuPipelineDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: sleep(t, 15),
				Config:    testAccCheckHerokuPipelinePromotionSingleTarget_basic(sourceApp, targetApp, pipelineName, pipelineOwnerID, appReleaseSlugID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuPipelineExists("heroku_pipeline.foobar-pipeline", &pipeline),
					testAccCheckHerokuAppReleaseExists("heroku_app_release.foobar-release", &release),
					testAccCheckHerokuPipelineCouplingExists("heroku_pipeline_coupling.foobar-source-pc", &pipelineCouplingSource),
					testAccCheckHerokuPipelineCouplingExists("heroku_pipeline_coupling.foobar-target-pc", &pipelineCouplingTarget),
					testAccCheckHerokuPipelinePromotionExists("heroku_pipeline_promotion.foobar-promotion", &promotion),
					resource.TestCheckResourceAttr("heroku_pipeline_promotion.foobar-promotion", "status", "succeeded"),
				),
			},
		},
	})
}

func testAccCheckHerokuPipelinePromotionSingleTarget_basic(sourceApp, targetApp, pipelineName, pipelineOwnerID, appReleaseSlugID string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar-source-app" {
	name = "%s"
	region = "us"
}
resource "heroku_app" "foobar-target-app" {
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
	app = "${heroku_app.foobar-source-app.name}"
	slug_id = "%s"
}
resource "heroku_pipeline_coupling" "foobar-source-pc" {
	app      = "${heroku_app.foobar-source-app.name}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "development"
}
resource "heroku_pipeline_coupling" "foobar-target-pc" {
	app      = "${heroku_app.foobar-target-app.name}"
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	stage    = "development"
}
resource "heroku_pipeline_promotion" "foobar-promotion" {
	pipeline = "${heroku_pipeline.foobar-pipeline.id}"
	source = "${heroku_app.foobar-source-app.name}"
	targets = ["${heroku_app.foobar-target-app.name}"]
}
`, sourceApp, targetApp, pipelineName, pipelineOwnerID, appReleaseSlugID)
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
