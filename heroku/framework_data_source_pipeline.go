package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Compile-time assertions that pipelineDataSource implements the required
// framework datasource interfaces.
var (
	_ datasource.DataSource              = (*pipelineDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*pipelineDataSource)(nil)
)

// NewPipelineDataSource returns the framework implementation of the
// heroku_pipeline data source.
func NewPipelineDataSource() datasource.DataSource {
	return &pipelineDataSource{}
}

type pipelineDataSource struct {
	config *Config
}

type pipelineDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	OwnerID   types.String `tfsdk:"owner_id"`
	OwnerType types.String `tfsdk:"owner_type"`
}

func (d *pipelineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (d *pipelineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"owner_id": schema.StringAttribute{
				Computed: true,
			},
			"owner_type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *pipelineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *pipelineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pipelineDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	pipeline, err := d.config.Api.PipelineInfo(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read pipeline %s", name),
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(pipeline.ID)
	data.Name = types.StringValue(pipeline.Name)

	if pipeline.Owner != nil {
		data.OwnerID = types.StringValue(pipeline.Owner.ID)
		data.OwnerType = types.StringValue(pipeline.Owner.Type)
	} else {
		data.OwnerID = types.StringValue("")
		data.OwnerType = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
