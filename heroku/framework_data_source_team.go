package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure interface compliance at compile time.
var (
	_ datasource.DataSource              = (*teamDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*teamDataSource)(nil)
)

// NewTeamDataSource returns the framework implementation of the heroku_team
// data source.
func NewTeamDataSource() datasource.DataSource {
	return &teamDataSource{}
}

type teamDataSource struct {
	config *Config
}

type teamDataSourceModel struct {
	ID                  types.String  `tfsdk:"id"`
	Name                types.String  `tfsdk:"name"`
	Default             types.Bool    `tfsdk:"default"`
	MembershipLimit     types.Float64 `tfsdk:"membership_limit"`
	ProvisionedLicenses types.Bool    `tfsdk:"provisioned_licenses"`
	Type                types.String  `tfsdk:"type"`
}

func (d *teamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *teamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"default": schema.BoolAttribute{
				Computed: true,
			},
			"membership_limit": schema.Float64Attribute{
				Computed: true,
			},
			"provisioned_licenses": schema.BoolAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *teamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *teamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model teamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := model.Name.ValueString()

	team, err := d.config.Api.TeamInfo(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read heroku_team %q", name),
			err.Error(),
		)
		return
	}

	model.ID = types.StringValue(team.ID)
	model.Name = types.StringValue(team.Name)
	model.Default = types.BoolValue(team.Default)
	model.ProvisionedLicenses = types.BoolValue(team.ProvisionedLicenses)
	model.Type = types.StringValue(team.Type)

	if team.MembershipLimit != nil {
		model.MembershipLimit = types.Float64Value(*team.MembershipLimit)
	} else {
		model.MembershipLimit = types.Float64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
