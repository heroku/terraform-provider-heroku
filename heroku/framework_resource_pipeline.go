package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*pipelineResource)(nil)
	_ resource.ResourceWithConfigure   = (*pipelineResource)(nil)
	_ resource.ResourceWithImportState = (*pipelineResource)(nil)
)

// NewPipelineResource returns the framework implementation of the
// heroku_pipeline resource.
func NewPipelineResource() resource.Resource {
	return &pipelineResource{}
}

type pipelineResource struct {
	config *Config
}

type pipelineOwnerModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

type pipelineResourceModel struct {
	ID    types.String         `tfsdk:"id"`
	Name  types.String         `tfsdk:"name"`
	Owner []pipelineOwnerModel `tfsdk:"owner"`
}

func (r *pipelineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *pipelineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				// TODO: port validation (StringMatch `^[a-z][a-z0-9-]{2,29}$`)
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							// TODO: port validation (IsUUID)
						},
						"type": schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							// TODO: port validation (StringInSlice ["team","user"])
						},
					},
				},
			},
		},
	}
}

func (r *pipelineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.config = config
}

func (r *pipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.PipelineCreateOpts{}

	name := plan.Name.ValueString()
	log.Printf("[DEBUG] New pipeline name: %s", name)
	opts.Name = name

	// If the owner is set, use it. Otherwise, pipeline ownership will default
	// to the authenticated user for this provider.
	opts.Owner = (*struct {
		ID   string `json:"id" url:"id,key"`
		Type string `json:"type" url:"type,key"`
	})(&struct {
		ID   string
		Type string
	}{ID: "", Type: ""})

	if len(plan.Owner) > 0 {
		opts.Owner.ID = plan.Owner[0].ID.ValueString()
		opts.Owner.Type = plan.Owner[0].Type.ValueString()
	} else {
		authUser, err := r.config.Api.AccountInfo(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error reading Heroku account", err.Error())
			return
		}
		opts.Owner.ID = authUser.ID
		opts.Owner.Type = "user"
	}

	log.Printf("[DEBUG] New pipeline owner id: %s", opts.Owner.ID)
	log.Printf("[DEBUG] New pipeline owner type: %s", opts.Owner.Type)
	log.Printf("[DEBUG] Pipeline create configuration: %#v", opts)

	p, err := r.config.Api.PipelineCreate(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating pipeline", err.Error())
		return
	}

	plan.ID = types.StringValue(p.ID)
	log.Printf("[INFO] Pipeline ID: %s", p.ID)

	r.setModelFromPipeline(&plan, p)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := r.config.Api.PipelineInfo(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving pipeline", err.Error())
		return
	}

	r.setModelFromPipeline(&state, p)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pipelineResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		opts := heroku.PipelineUpdateOpts{
			Name: &name,
		}

		_, err := r.config.Api.PipelineUpdate(ctx, state.ID.ValueString(), opts)
		if err != nil {
			resp.Diagnostics.AddError("Error updating pipeline", err.Error())
			return
		}
	}

	p, err := r.config.Api.PipelineInfo(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving pipeline", err.Error())
		return
	}

	plan.ID = state.ID
	r.setModelFromPipeline(&plan, p)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pipelineResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting pipeline: %s", state.ID.ValueString())

	_, err := r.config.Api.PipelineDelete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting pipeline", fmt.Sprintf("Error deleting pipeline: %s", err))
		return
	}
}

func (r *pipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	p, err := r.config.Api.PipelineInfo(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing pipeline", err.Error())
		return
	}

	state := pipelineResourceModel{}
	state.ID = types.StringValue(p.ID)
	r.setModelFromPipeline(&state, p)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pipelineResource) setModelFromPipeline(m *pipelineResourceModel, p *heroku.Pipeline) {
	m.Name = types.StringValue(p.Name)

	if p.Owner != nil {
		m.Owner = []pipelineOwnerModel{
			{
				ID:   types.StringValue(p.Owner.ID),
				Type: types.StringValue(p.Owner.Type),
			},
		}
	} else {
		m.Owner = []pipelineOwnerModel{}
	}
}
