package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*pipelineCouplingResource)(nil)
	_ resource.ResourceWithConfigure    = (*pipelineCouplingResource)(nil)
	_ resource.ResourceWithImportState  = (*pipelineCouplingResource)(nil)
	_ resource.ResourceWithUpgradeState = (*pipelineCouplingResource)(nil)
)

// NewPipelineCouplingResource returns the framework implementation of the
// heroku_pipeline_coupling resource.
func NewPipelineCouplingResource() resource.Resource {
	return &pipelineCouplingResource{}
}

type pipelineCouplingResource struct {
	config *Config
}

// pipelineCouplingResourceModel is the model for schema version 1 (current).
type pipelineCouplingResourceModel struct {
	ID       types.String `tfsdk:"id"`
	AppID    types.String `tfsdk:"app_id"`
	Pipeline types.String `tfsdk:"pipeline"`
	Stage    types.String `tfsdk:"stage"`
}

// pipelineCouplingResourceModelV0 mirrors resourceHerokuPipelineCouplingV0 —
// the v0 schema used "app" (name or UUID) instead of "app_id".
type pipelineCouplingResourceModelV0 struct {
	ID       types.String `tfsdk:"id"`
	App      types.String `tfsdk:"app"`
	Pipeline types.String `tfsdk:"pipeline"`
	Stage    types.String `tfsdk:"stage"`
}

func (r *pipelineCouplingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_coupling"
}

func (r *pipelineCouplingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"pipeline": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"stage": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					stringvalidator.OneOf("review", "development", "staging", "production"),
				},
			},
		},
	}
}

func (r *pipelineCouplingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pipelineCouplingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pipelineCouplingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.PipelineCouplingCreateOpts{
		App:      plan.AppID.ValueString(),
		Pipeline: plan.Pipeline.ValueString(),
		Stage:    plan.Stage.ValueString(),
	}

	log.Printf("[DEBUG] PipelineCoupling create configuration: %#v", opts)

	p, err := r.config.Api.PipelineCouplingCreate(ctx, opts)
	if err != nil {
		// Enhance generation-related errors with app context.
		errMsg := err.Error()
		if strings.Contains(errMsg, "same generation") {
			if app, appErr := r.config.Api.AppInfo(ctx, plan.AppID.ValueString()); appErr == nil {
				resp.Diagnostics.AddError(
					"Error creating pipeline coupling",
					fmt.Sprintf("%s\n\nYour app '%s' is %s generation. Ensure all apps in the pipeline use the same generation (Cedar or Fir)",
						errMsg, app.Name, app.Generation.Name),
				)
				return
			}
		}
		resp.Diagnostics.AddError("Error creating pipeline coupling", fmt.Sprintf("error creating pipeline: %s", err))
		return
	}

	plan.ID = types.StringValue(p.ID)
	log.Printf("[INFO] PipelineCoupling ID: %s", p.ID)

	if err := r.readPipelineCoupling(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline coupling after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelineCouplingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pipelineCouplingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readPipelineCoupling(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline coupling", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op: all writable fields are ForceNew, so Terraform replaces
// the resource instead of calling Update. This implementation is required to
// satisfy resource.Resource.
func (r *pipelineCouplingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are ForceNew; this method should never be called.
	var plan pipelineCouplingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelineCouplingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pipelineCouplingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting pipeline coupling: %s", state.ID.ValueString())

	_, err := r.config.Api.PipelineCouplingDelete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting pipeline coupling", fmt.Sprintf("error deleting pipeline: %s", err))
		return
	}
}

func (r *pipelineCouplingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// UpgradeState implements resource.ResourceWithUpgradeState and mirrors the
// SDKv2 StateUpgraders: v0 used "app" (name or UUID); v1 uses "app_id" (UUID).
func (r *pipelineCouplingResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"app": schema.StringAttribute{
						Required: true,
					},
					"pipeline": schema.StringAttribute{
						Required: true,
					},
					"stage": schema.StringAttribute{
						Required: true,
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior pipelineCouplingResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading pipeline coupling state", err.Error())
					return
				}

				upgraded := pipelineCouplingResourceModel{
					ID:       prior.ID,
					AppID:    types.StringValue(appID),
					Pipeline: prior.Pipeline,
					Stage:    prior.Stage,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// readPipelineCoupling fetches the current coupling from the API and populates m.
func (r *pipelineCouplingResource) readPipelineCoupling(ctx context.Context, m *pipelineCouplingResourceModel) error {
	p, err := r.config.Api.PipelineCouplingInfo(ctx, m.ID.ValueString())
	if err != nil {
		return fmt.Errorf("error retrieving pipeline: %s", err)
	}

	m.AppID = types.StringValue(p.App.ID)
	m.Stage = types.StringValue(p.Stage)
	m.Pipeline = types.StringValue(p.Pipeline.ID)

	return nil
}
