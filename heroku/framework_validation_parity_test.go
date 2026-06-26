package heroku

import (
	"fmt"
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

// TestAccFrameworkParity_AppRelease verifies the restored SDKv2 constraints on
// heroku_app_release: app_id IsUUID, the slug_id/oci_image exactly-one rule
// (SDKv2 ConflictsWith + AtLeastOneOf), and validateOCIImage on oci_image. All
// cases fail at plan time, so no API calls are made.
func TestAccFrameworkParity_AppRelease(t *testing.T) {
	const validUUID = "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		name   string
		config string
		err    *regexp.Regexp
	}{
		{
			name: "invalid_app_id",
			config: fmt.Sprintf(`
resource "heroku_app_release" "t" {
  app_id  = "not-a-uuid"
  slug_id = "%s"
}`, validUUID),
			err: regexp.MustCompile(`to be a valid UUID`),
		},
		{
			name: "both_sources",
			config: fmt.Sprintf(`
resource "heroku_app_release" "t" {
  app_id    = "%s"
  slug_id   = "%s"
  oci_image = "%s"
}`, validUUID, validUUID, validUUID),
			err: regexp.MustCompile(`one \(and only one\) of`),
		},
		{
			name: "no_source",
			config: fmt.Sprintf(`
resource "heroku_app_release" "t" {
  app_id = "%s"
}`, validUUID),
			err: regexp.MustCompile(`one \(and only one\) of`),
		},
		{
			name: "invalid_oci_image",
			config: fmt.Sprintf(`
resource "heroku_app_release" "t" {
  app_id    = "%s"
  oci_image = "not-a-valid-image"
}`, validUUID),
			err: regexp.MustCompile(`invalid OCI image identifier`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: tc.err,
					},
				},
			})
		})
	}
}

// TestAccFrameworkParity_Cardinality verifies restored SDKv2 MinItems/MaxItems
// (and StringIsNotEmpty) constraints: team_collaborator.permissions (1..4),
// heroku_app.organization (MaxItems 1) + organization.name (non-empty),
// app_webhook.include (MinItems 1), telemetry_drain.signals (MinItems 1), and
// review_app_config.deploy_target (MaxItems 1). All cases fail at plan time.
func TestAccFrameworkParity_Cardinality(t *testing.T) {
	const u = "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		name   string
		config string
		err    *regexp.Regexp
	}{
		{
			name: "team_collaborator_permissions_empty",
			config: fmt.Sprintf(`
resource "heroku_team_collaborator" "t" {
  app_id      = "%s"
  email       = "collaborator@example.com"
  permissions = []
}`, u),
			err: regexp.MustCompile(`must contain at least`),
		},
		{
			name: "team_collaborator_permissions_too_many",
			config: fmt.Sprintf(`
resource "heroku_team_collaborator" "t" {
  app_id      = "%s"
  email       = "collaborator@example.com"
  permissions = ["a", "b", "c", "d", "e"]
}`, u),
			err: regexp.MustCompile(`at most 4`),
		},
		{
			name: `app_organization_too_many`,
			config: `
resource "heroku_app" "t" {
  name   = "tftest-app"
  region = "us"
  organization {
    name = "org-a"
  }
  organization {
    name = "org-b"
  }
}`,
			err: regexp.MustCompile(`must contain at most`),
		},
		{
			name: `app_organization_empty_name`,
			config: `
resource "heroku_app" "t" {
  name   = "tftest-app"
  region = "us"
  organization {
    name = ""
  }
}`,
			err: regexp.MustCompile(`length must be at least`),
		},
		{
			name: "app_webhook_include_empty",
			config: fmt.Sprintf(`
resource "heroku_app_webhook" "t" {
  app_id  = "%s"
  level   = "notify"
  url     = "https://example.com/hook"
  include = []
}`, u),
			err: regexp.MustCompile(`must contain at least`),
		},
		{
			name: "telemetry_drain_signals_empty",
			config: fmt.Sprintf(`
resource "heroku_telemetry_drain" "t" {
  owner_id      = "%s"
  owner_type    = "app"
  exporter_type = "otlp"
  endpoint      = "https://example.com"
  signals       = []
  headers       = {}
}`, u),
			err: regexp.MustCompile(`must contain at least`),
		},
		{
			name: "review_app_config_deploy_target_too_many",
			config: fmt.Sprintf(`
resource "heroku_review_app_config" "t" {
  pipeline_id = "%s"
  org_repo    = "acme/app"
  deploy_target {
    id   = "us"
    type = "region"
  }
  deploy_target {
    id   = "%s"
    type = "space"
  }
}`, u, u),
			err: regexp.MustCompile(`must contain at most`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: tc.err,
					},
				},
			})
		})
	}
}

// TestAccFrameworkParity_SpaceInboundRuleset verifies restored SDKv2
// constraints on heroku_space_inbound_ruleset: the rule block MinItems 1
// (SizeAtLeast) and validation.IsCIDRNetwork(0, 32) on rule.source. Both cases
// fail at plan time.
func TestAccFrameworkParity_SpaceInboundRuleset(t *testing.T) {
	cases := []struct {
		name   string
		config string
		err    *regexp.Regexp
	}{
		{
			name: "no_rules",
			config: `
resource "heroku_space_inbound_ruleset" "t" {
  space = "tftest-space"
}`,
			err: regexp.MustCompile(`must contain at least`),
		},
		{
			name: "invalid_cidr",
			config: `
resource "heroku_space_inbound_ruleset" "t" {
  space = "tftest-space"
  rule {
    action = "allow"
    source = "not-a-cidr"
  }
}`,
			err: regexp.MustCompile(`valid CIDR network`),
		},
		{
			name: "non_network_cidr",
			config: `
resource "heroku_space_inbound_ruleset" "t" {
  space = "tftest-space"
  rule {
    action = "allow"
    source = "10.0.0.1/8"
  }
}`,
			err: regexp.MustCompile(`valid network value`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: tc.err,
					},
				},
			})
		})
	}
}

// TestAccFrameworkParity_PipelineAndSpace verifies restored SDKv2 constraints
// on heroku_pipeline_config_var (pipeline_id IsUUID, pipeline_stage enum),
// heroku_pipeline_promotion (IsUUID on pipeline/source_app_id/release_id and
// each targets element), and heroku_space (generation enum). All cases fail at
// plan time.
func TestAccFrameworkParity_PipelineAndSpace(t *testing.T) {
	const u = "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		name   string
		config string
		err    *regexp.Regexp
	}{
		{
			name: "pipeline_config_var_invalid_pipeline_id",
			config: `
resource "heroku_pipeline_config_var" "t" {
  pipeline_id    = "not-a-uuid"
  pipeline_stage = "test"
}`,
			err: regexp.MustCompile(`to be a valid UUID`),
		},
		{
			name: "pipeline_config_var_invalid_stage",
			config: fmt.Sprintf(`
resource "heroku_pipeline_config_var" "t" {
  pipeline_id    = "%s"
  pipeline_stage = "bogus"
}`, u),
			err: regexp.MustCompile(`value must be one of`),
		},
		{
			name: "pipeline_promotion_invalid_pipeline",
			config: fmt.Sprintf(`
resource "heroku_pipeline_promotion" "t" {
  pipeline      = "not-a-uuid"
  source_app_id = "%s"
  release_id    = "%s"
  targets       = ["%s"]
}`, u, u, u),
			err: regexp.MustCompile(`to be a valid UUID`),
		},
		{
			name: "pipeline_promotion_invalid_target",
			config: fmt.Sprintf(`
resource "heroku_pipeline_promotion" "t" {
  pipeline      = "%s"
  source_app_id = "%s"
  release_id    = "%s"
  targets       = ["not-a-uuid"]
}`, u, u, u),
			err: regexp.MustCompile(`to be a valid UUID`),
		},
		{
			name: "space_invalid_generation",
			config: `
resource "heroku_space" "t" {
  name         = "tftest-space"
  organization = "tftest"
  generation   = "bogus"
}`,
			err: regexp.MustCompile(`value must be one of`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: tc.err,
					},
				},
			})
		})
	}
}
