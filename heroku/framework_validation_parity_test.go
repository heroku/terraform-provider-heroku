package heroku

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFrameworkParity_AppIDUUID verifies that the SDKv2 validation.IsUUID
// constraint on each resource's app_id attribute was restored after the
// terraform-plugin-framework migration. Each case fails at plan time (config
// validation), so no API calls are made and no real app is required.
func TestAccFrameworkParity_AppIDUUID(t *testing.T) {
	cases := []struct {
		name   string
		config string
	}{
		{
			name: "addon",
			config: `
resource "heroku_addon" "t" {
  app_id = "not-a-uuid"
  plan   = "heroku-postgresql:essential-0"
}`,
		},
		{
			name: "addon_attachment",
			config: `
resource "heroku_addon_attachment" "t" {
  app_id   = "not-a-uuid"
  addon_id = "00000000-0000-0000-0000-000000000000"
}`,
		},
		{
			name: "app_feature",
			config: `
resource "heroku_app_feature" "t" {
  app_id  = "not-a-uuid"
  name    = "some-feature"
  enabled = true
}`,
		},
		{
			name: "build",
			config: `
resource "heroku_build" "t" {
  app_id = "not-a-uuid"
  source {
    url = "https://example.com/source.tgz"
  }
}`,
		},
		{
			name: "collaborator",
			config: `
resource "heroku_collaborator" "t" {
  app_id = "not-a-uuid"
  email  = "collaborator@example.com"
}`,
		},
		{
			name: "domain",
			config: `
resource "heroku_domain" "t" {
  app_id   = "not-a-uuid"
  hostname = "terraform.example.com"
}`,
		},
		{
			name: "drain",
			config: `
resource "heroku_drain" "t" {
  app_id = "not-a-uuid"
  url    = "syslog://example.com:514"
}`,
		},
		{
			name: "formation",
			config: `
resource "heroku_formation" "t" {
  app_id   = "not-a-uuid"
  type     = "web"
  quantity = 1
  size     = "Standard-1X"
}`,
		},
		{
			name: "slug",
			config: `
resource "heroku_slug" "t" {
  app_id                          = "not-a-uuid"
  file_path                       = "slug.tgz"
  process_types = {
    web = "ruby server.rb"
  }
}`,
		},
		{
			name: "team_collaborator",
			config: `
resource "heroku_team_collaborator" "t" {
  app_id      = "not-a-uuid"
  email       = "collaborator@example.com"
  permissions = ["view"]
}`,
		},
	}

	uuidErr := regexp.MustCompile(`to be a valid UUID`)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: uuidErr,
					},
				},
			})
		})
	}
}
