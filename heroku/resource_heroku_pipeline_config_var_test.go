package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccHerokuPipelineConfigVar_TestStage_Basic(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	stage := "test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipelineConfigVar_basic(name, stage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "vars.ENV", "test"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "vars.TARGET", "develop"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "sensitive_vars.TEST_ACCESS_TOKEN", "some_access token"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "all_vars.ENV", "test"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "all_vars.TARGET", "develop"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "all_vars.TEST_ACCESS_TOKEN", "some_access token"),
				),
			},
		},
	})
}

func TestAccHerokuPipelineConfigVar_ReviewStage_Basic(t *testing.T) {
	name := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	stage := "review"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuPipelineConfigVar_basic(name, stage),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "vars.ENV", "test"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "vars.TARGET", "develop"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "sensitive_vars.TEST_ACCESS_TOKEN", "some_access token"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "all_vars.ENV", "test"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "all_vars.TARGET", "develop"),
					resource.TestCheckResourceAttr(
						"heroku_pipeline_config_var.configs", "all_vars.TEST_ACCESS_TOKEN", "some_access token"),
				),
			},
		},
	})
}

func testAccCheckHerokuPipelineConfigVar_basic(name, stage string) string {
	return fmt.Sprintf(`
resource "heroku_pipeline" "foobar" {
  name = "%s"
}

resource "heroku_pipeline_config_var" "configs" {
  pipeline_id = heroku_pipeline.foobar.id
  pipeline_stage = "%s"

  vars = {
    ENV = "test"
    TARGET = "develop"
  }

  sensitive_vars = {
    TEST_ACCESS_TOKEN = "some_access token"
  }
}
`, name, stage)
}
