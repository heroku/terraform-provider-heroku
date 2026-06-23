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
	_ datasource.DataSource              = (*spacePeeringInfoDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*spacePeeringInfoDataSource)(nil)
)

// NewSpacePeeringInfoDataSource returns the framework implementation of the
// heroku_space_peering_info data source.
func NewSpacePeeringInfoDataSource() datasource.DataSource {
	return &spacePeeringInfoDataSource{}
}

type spacePeeringInfoDataSource struct {
	config *Config
}

type spacePeeringInfoDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	AwsAccountID          types.String `tfsdk:"aws_account_id"`
	AwsRegion             types.String `tfsdk:"aws_region"`
	VpcID                 types.String `tfsdk:"vpc_id"`
	VpcCIDR               types.String `tfsdk:"vpc_cidr"`
	DynoCIDRBlocks        types.List   `tfsdk:"dyno_cidr_blocks"`
	UnavailableCIDRBlocks types.List   `tfsdk:"unavailable_cidr_blocks"`
}

func (d *spacePeeringInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_peering_info"
}

func (d *spacePeeringInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"aws_account_id": schema.StringAttribute{
				Computed: true,
			},
			"aws_region": schema.StringAttribute{
				Computed: true,
			},
			"vpc_id": schema.StringAttribute{
				Computed: true,
			},
			"vpc_cidr": schema.StringAttribute{
				Computed: true,
			},
			"dyno_cidr_blocks": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"unavailable_cidr_blocks": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *spacePeeringInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *spacePeeringInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model spacePeeringInfoDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := model.Name.ValueString()

	peeringInfo, err := d.config.Api.PeeringInfoInfo(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to retrieve peering info for space %s", name),
			err.Error(),
		)
		return
	}

	model.ID = types.StringValue(name)
	model.AwsAccountID = types.StringValue(peeringInfo.AwsAccountID)
	model.AwsRegion = types.StringValue(peeringInfo.AwsRegion)
	model.VpcID = types.StringValue(peeringInfo.VpcID)
	model.VpcCIDR = types.StringValue(peeringInfo.VpcCIDR)

	dynoCIDRList, diags := types.ListValueFrom(ctx, types.StringType, peeringInfo.DynoCIDRBlocks)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.DynoCIDRBlocks = dynoCIDRList

	unavailableCIDRList, diags := types.ListValueFrom(ctx, types.StringType, peeringInfo.UnavailableCIDRBlocks)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.UnavailableCIDRBlocks = unavailableCIDRList

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
