package heroku

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHerokuCollaborator_importBasic(t *testing.T) {
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))
	testUser := testAccConfig.GetNonAdminUserOrAbort(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCollaborator_Basic(appName, testUser),
			},
			{
				ResourceName:            "heroku_collaborator.foobar-collaborator",
				ImportStateId:           buildCompositeID(appName, testUser),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"suppress_invites"},
			},
		},
	})
}
