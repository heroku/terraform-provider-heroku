package heroku

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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
		Providers: testAccProviders,
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
