package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/heroku/heroku-go/v3"
	"testing"
)

func TestAccHerokuAccountFeature_Basic(t *testing.T) {
	var accountFeature heroku.AccountFeature
	featureName := "app-overview"
	enabled := true

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuAccountFeatureConfig_Basic(featureName, enabled),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuAccountFeatureStatus("heroku_account_feature.foobar", &accountFeature),
					resource.TestCheckResourceAttr("heroku_account_feature.foobar", "enabled", "true"),
				),
			},
		},
	})
}

func testAccCheckHerokuAccountFeatureStatus(n string, accountFeature *heroku.AccountFeature) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No account feature id set")
		}

		client := testAccProvider.Meta().(*Config).Api

		foundAccountFeature, err := client.AccountFeatureInfo(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if foundAccountFeature.ID != rs.Primary.ID {
			return fmt.Errorf("Account Feature not found")
		}

		*accountFeature = *foundAccountFeature

		return nil
	}
}

func testAccCheckHerokuAccountFeatureConfig_Basic(featureName string, enabled bool) string {
	return fmt.Sprintf(`
resource "heroku_account_feature" "foobar" {
	name = %s
	enabled = %v
}
`, featureName, enabled)
}
