package heroku

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

// frameworkProvider is the terraform-plugin-framework implementation of the
// Heroku provider. It is the sole provider implementation served by main.go
// (protocol 6) following the full migration off terraform-plugin-sdk/v2.
type frameworkProvider struct{}

// NewFrameworkProvider returns the framework provider implementation served by
// the provider server.
func NewFrameworkProvider() provider.Provider {
	return &frameworkProvider{}
}

var _ provider.Provider = (*frameworkProvider)(nil)

func (p *frameworkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "heroku"
}

func (p *frameworkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				Optional: true,
			},
			"headers": schema.StringAttribute{
				Optional: true,
			},
			"url": schema.StringAttribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"customizations": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"set_app_all_config_vars_in_state": schema.BoolAttribute{
							Optional: true,
						},
						"set_addon_config_vars_in_state": schema.BoolAttribute{
							Optional: true,
						},
					},
				},
			},
			"delays": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"post_app_create_delay": schema.Int64Attribute{
							Optional: true,
						},
						"post_space_create_delay": schema.Int64Attribute{
							Optional: true,
						},
						"post_domain_create_delay": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
			"timeouts": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"addon_create_timeout": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

// frameworkProviderModel mirrors the provider-configuration schema.
type frameworkProviderModel struct {
	Email          types.String `tfsdk:"email"`
	APIKey         types.String `tfsdk:"api_key"`
	Headers        types.String `tfsdk:"headers"`
	URL            types.String `tfsdk:"url"`
	Customizations types.List   `tfsdk:"customizations"`
	Delays         types.List   `tfsdk:"delays"`
	Timeouts       types.List   `tfsdk:"timeouts"`
}

type frameworkCustomizationsModel struct {
	SetAppAllConfigVarsInState types.Bool `tfsdk:"set_app_all_config_vars_in_state"`
	SetAddonConfigVarsInState  types.Bool `tfsdk:"set_addon_config_vars_in_state"`
}

type frameworkDelaysModel struct {
	PostAppCreateDelay    types.Int64 `tfsdk:"post_app_create_delay"`
	PostSpaceCreateDelay  types.Int64 `tfsdk:"post_space_create_delay"`
	PostDomainCreateDelay types.Int64 `tfsdk:"post_domain_create_delay"`
}

type frameworkTimeoutsModel struct {
	AddonCreateTimeout types.Int64 `tfsdk:"addon_create_timeout"`
}

func (p *frameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data frameworkProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config := NewConfig()

	// headers: config value, else HEROKU_HEADERS env (mirrors SDKv2 DefaultFunc).
	headersJSON := stringOrEnv(data.Headers, "HEROKU_HEADERS", "")
	if headersJSON != "" {
		headers := make(map[string]string)
		if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("headers"),
				"Invalid headers",
				fmt.Sprintf("Could not parse headers as JSON: %s", err),
			)
			return
		}
		for k, v := range headers {
			config.Headers.Set(k, v)
		}
	}

	// url: config value, else HEROKU_API_URL env, else heroku.DefaultURL.
	config.URL = stringOrEnv(data.URL, "HEROKU_API_URL", heroku.DefaultURL)

	// customizations block.
	if !data.Customizations.IsNull() && !data.Customizations.IsUnknown() {
		var blocks []frameworkCustomizationsModel
		resp.Diagnostics.Append(data.Customizations.ElementsAs(ctx, &blocks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(blocks) > 1 {
			resp.Diagnostics.AddError("Provider configuration error", "only one customizations block is permitted")
			return
		}
		for _, b := range blocks {
			if !b.SetAppAllConfigVarsInState.IsNull() {
				config.SetAppAllConfigVarsInState = b.SetAppAllConfigVarsInState.ValueBool()
			}
			if !b.SetAddonConfigVarsInState.IsNull() {
				config.SetAddonConfigVarsInState = b.SetAddonConfigVarsInState.ValueBool()
			}
		}
	}

	// delays block.
	if !data.Delays.IsNull() && !data.Delays.IsUnknown() {
		var blocks []frameworkDelaysModel
		resp.Diagnostics.Append(data.Delays.ElementsAs(ctx, &blocks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(blocks) > 1 {
			resp.Diagnostics.AddError("Provider configuration error", "only one delays block is permitted")
			return
		}
		for _, b := range blocks {
			if !b.PostAppCreateDelay.IsNull() {
				config.PostAppCreateDelay = b.PostAppCreateDelay.ValueInt64()
			}
			if !b.PostSpaceCreateDelay.IsNull() {
				config.PostSpaceCreateDelay = b.PostSpaceCreateDelay.ValueInt64()
			}
			if !b.PostDomainCreateDelay.IsNull() {
				config.PostDomainCreateDelay = b.PostDomainCreateDelay.ValueInt64()
			}
		}
	}

	// timeouts block.
	if !data.Timeouts.IsNull() && !data.Timeouts.IsUnknown() {
		var blocks []frameworkTimeoutsModel
		resp.Diagnostics.Append(data.Timeouts.ElementsAs(ctx, &blocks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(blocks) > 1 {
			resp.Diagnostics.AddError("provider configuration error", "only one timeouts block is permitted")
			return
		}
		for _, b := range blocks {
			if !b.AddonCreateTimeout.IsNull() {
				config.AddonCreateTimeout = b.AddonCreateTimeout.ValueInt64()
			}
		}
	}

	if err := config.applyNetrcFile(); err != nil {
		resp.Diagnostics.AddError("Error reading netrc file", err.Error())
		return
	}

	// The provider configuration takes precedence over netrc. Mirror the SDKv2
	// DefaultFunc behaviour: config value, else environment variable.
	if email := stringOrEnv(data.Email, "HEROKU_EMAIL", ""); email != "" {
		config.Email = email
	}
	if apiKey := stringOrEnv(data.APIKey, "HEROKU_API_KEY", ""); apiKey != "" {
		config.APIKey = apiKey
	}

	if err := config.initializeAPI(); err != nil {
		resp.Diagnostics.AddError("Error initializing Heroku API client", err.Error())
		return
	}

	resp.ResourceData = config
	resp.DataSourceData = config
}

// Resources returns the framework-implemented managed resources. Resources are
// added here as they are migrated off the SDKv2 Provider().
func (p *frameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAccountFeatureResource,
		NewAddonResource,
		NewAddonAttachmentResource,
		NewAppConfigAssociationResource,
		NewAppFeatureResource,
		NewAppReleaseResource,
		NewAppResource,
		NewAppWebhookResource,
		NewBuildResource,
		NewCollaboratorResource,
		NewConfigResource,
		NewDomainResource,
		NewDrainResource,
		NewFormationResource,
		NewPipelineResource,
		NewPipelineConfigVarResource,
		NewPipelineCouplingResource,
		NewPipelinePromotionResource,
		NewReviewAppConfigResource,
		NewSlugResource,
		NewSpaceResource,
		NewSpaceAppAccessResource,
		NewSpaceInboundRulesetResource,
		NewSpacePeeringConnectionAccepterResource,
		NewSpaceVPNConnectionResource,
		NewSSLResource,
		NewTeamCollaboratorResource,
		NewTeamMemberResource,
		NewTelemetryDrainResource,
	}
}

// DataSources returns the framework-implemented data sources. Data sources are
// added here as they are migrated off the SDKv2 Provider().
func (p *frameworkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAddonDataSource,
		NewAppDataSource,
		NewPipelineDataSource,
		NewSpaceDataSource,
		NewSpacePeeringInfoDataSource,
		NewTeamDataSource,
		NewTeamMembersDataSource,
	}
}

// stringOrEnv returns the configured value if known and non-null, otherwise the
// value of the named environment variable, otherwise fallback.
func stringOrEnv(v types.String, envKey, fallback string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	if env := os.Getenv(envKey); env != "" {
		return env
	}
	return fallback
}
