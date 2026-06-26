package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Compile-time interface assertions.
var (
	_ datasource.DataSource              = (*addonDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*addonDataSource)(nil)
)

// NewAddonDataSource returns the framework implementation of the heroku_addon
// data source.
func NewAddonDataSource() datasource.DataSource {
	return &addonDataSource{}
}

type addonDataSource struct {
	config *Config
}

type addonDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	AppID      types.String `tfsdk:"app_id"`
	Plan       types.String `tfsdk:"plan"`
	ProviderID types.String `tfsdk:"provider_id"`
	ConfigVars types.List   `tfsdk:"config_vars"`
}

func (d *addonDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_addon"
}

func (d *addonDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"app_id": schema.StringAttribute{
				Computed: true,
			},
			"plan": schema.StringAttribute{
				Computed: true,
			},
			"provider_id": schema.StringAttribute{
				Computed: true,
			},
			"config_vars": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *addonDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *Config, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.config = config
}

func (d *addonDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model addonDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := model.Name.ValueString()

	addon, err := resourceHerokuAddonRetrieve(name, d.config.Api)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to retrieve addon %s", name),
			err.Error(),
		)
		return
	}

	model.ID = types.StringValue(addon.ID)
	model.Name = types.StringValue(addon.Name)
	model.AppID = types.StringValue(addon.App.ID)
	model.Plan = types.StringValue(addon.Plan.Name)
	model.ProviderID = types.StringValue(addon.ProviderID)

	configVarsVal, diags := types.ListValueFrom(ctx, types.StringType, addon.ConfigVars)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.ConfigVars = configVarsVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
