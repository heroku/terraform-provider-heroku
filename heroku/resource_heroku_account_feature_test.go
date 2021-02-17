package heroku

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
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
					testAccCheckHerokuAccountFeatureDescription(&accountFeature, "An overview page for an app"),
				),
			},
		},
	})
}

func testAccCheckHerokuAccountFeatureDescription(accountFeature *heroku.AccountFeature, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if accountFeature.Description != n {
			return fmt.Errorf("accountFeature's description is not correct. Found: %s | Got: %s", accountFeature.Description, n)
		}

		return nil
	}
}

func testAccCheckHerokuAccountFeatureDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Config).Api

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_account_feature" {
			continue
		}

		_, err := client.AccountFeatureInfo(context.TODO(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Account Feature still exists")
		}
	}

	return nil
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
		accountEmail, accountFeatureName, _ := parseCompositeID(rs.Primary.ID)

		// Check to make sure accountEmail matches what was set as the resource Id
		account, err := client.AccountInfo(context.TODO())
		if err != nil {
			return err
		}

		if account.Email != accountEmail {
			return fmt.Errorf("The email address in the resource Id " +
				"does not match the address of the API TOKEN used in this test.")
		}

		foundAccountFeature, err := client.AccountFeatureInfo(context.TODO(), accountFeatureName)
		if err != nil {
			return err
		}

		if foundAccountFeature.Name != accountFeatureName {
			return fmt.Errorf("Account Feature not found")
		}

		*accountFeature = *foundAccountFeature

		return nil
	}
}

func testAccCheckHerokuAccountFeatureConfig_Basic(featureName string, enabled bool) string {
	return fmt.Sprintf(`
resource "heroku_account_feature" "foobar" {
	name = "%s"
	enabled = %v
}
`, featureName, enabled)
}
