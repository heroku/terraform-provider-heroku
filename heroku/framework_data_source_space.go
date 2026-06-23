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
	_ datasource.DataSource              = (*spaceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*spaceDataSource)(nil)
)

// NewSpaceDataSource returns the framework implementation of the heroku_space
// data source.
func NewSpaceDataSource() datasource.DataSource {
	return &spaceDataSource{}
}

type spaceDataSource struct {
	config *Config
}

type spaceDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Organization types.String `tfsdk:"organization"`
	UUID         types.String `tfsdk:"uuid"`
	CIDR         types.String `tfsdk:"cidr"`
	DataCIDR     types.String `tfsdk:"data_cidr"`
	OutboundIPs  types.List   `tfsdk:"outbound_ips"`
	Region       types.String `tfsdk:"region"`
	State        types.String `tfsdk:"state"`
	Shield       types.Bool   `tfsdk:"shield"`
	Generation   types.String `tfsdk:"generation"`
}

func (d *spaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space"
}

func (d *spaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"organization": schema.StringAttribute{
				Computed: true,
			},
			"uuid": schema.StringAttribute{
				Computed: true,
			},
			"cidr": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"data_cidr": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"outbound_ips": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"region": schema.StringAttribute{
				Computed: true,
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
			"shield": schema.BoolAttribute{
				Computed: true,
			},
			"generation": schema.StringAttribute{
				Computed:    true,
				Description: "Generation of the space platform (cedar or fir)",
			},
		},
	}
}

func (d *spaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *spaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model spaceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := model.Name.ValueString()

	// Fetch space info via SpaceStateRefreshFunc which also retrieves NAT outbound
	// IPs (populating spaceWithNAT.NAT.Sources). This mirrors the SDK data-source
	// read which calls the same function.
	spaceRaw, _, err := SpaceStateRefreshFunc(d.config.Api, name)()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read heroku_space %q", name),
			err.Error(),
		)
		return
	}

	space := spaceRaw.(*spaceWithNAT)

	// Populate model fields mirroring resourceHerokuSpaceRead.
	model.ID = types.StringValue(name)
	model.UUID = types.StringValue(space.ID)
	model.Name = types.StringValue(space.Name)
	model.Organization = types.StringValue(space.Organization.Name)
	model.Region = types.StringValue(space.Region.Name)
	model.State = types.StringValue(space.State)
	model.Shield = types.BoolValue(space.Shield)
	model.CIDR = types.StringValue(space.CIDR)
	model.DataCIDR = types.StringValue(space.DataCIDR)
	model.Generation = types.StringValue(space.Generation.Name)

	// outbound_ips: convert []string from NAT.Sources into a framework List.
	outboundIPsVal, diags := types.ListValueFrom(ctx, types.StringType, space.NAT.Sources)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.OutboundIPs = outboundIPsVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
