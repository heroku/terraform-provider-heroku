package heroku

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestAccHerokuAccountFeature_importBasic(t *testing.T) {
	accountEmail := testAccConfig.GetEmailOrSkip(t)
	featureName := "app-overview"
	enabled := false

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAccountFeatureConfig_Basic(featureName, enabled),
			},
			{
				ResourceName:      "heroku_account_feature.foobar",
				ImportStateId:     buildCompositeID(accountEmail, featureName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
