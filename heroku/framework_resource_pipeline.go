package heroku

import (
	"context"
	"fmt"
	"log"
	"regexp"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

// pipelineNameRegex mirrors the SDKv2 validation.StringMatch regex for the
// pipeline name: a lowercase letter followed by 2-29 lowercase letters, digits
// or dashes (3-30 characters total).
var pipelineNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{2,29}$`)

// pipelineOwnerIDValidator validates that owner.id is a UUID, mirroring the
// SDKv2 validation.IsUUID ValidateFunc. The message reproduces the SDKv2 path
// format ("owner.0.id") because the acceptance test matches it literally.
type pipelineOwnerIDValidator struct{}

func (v pipelineOwnerIDValidator) Description(_ context.Context) string {
	return "value must be a valid UUID"
}

func (v pipelineOwnerIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v pipelineOwnerIDValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if _, err := uuid.ParseUUID(val); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid UUID",
			fmt.Sprintf(`expected "owner.0.id" to be a valid UUID, got %s`, val),
		)
	}
}

// pipelineOwnerTypeValidator validates that owner.type is "team" or "user",
// mirroring the SDKv2 validation.StringInSlice ValidateFunc. The message
// reproduces the SDKv2 path format ("owner.0.type") for the acceptance test.
type pipelineOwnerTypeValidator struct{}

func (v pipelineOwnerTypeValidator) Description(_ context.Context) string {
	return `value must be one of "team" or "user"`
}

func (v pipelineOwnerTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v pipelineOwnerTypeValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if val != "team" && val != "user" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid owner type",
			fmt.Sprintf(`expected owner.0.type to be one of ["team" "user"], got %s`, val),
		)
	}
}

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

// pipelineOwnerObjectType returns the object type for an "owner" list element,
// matching the nested attribute schema.
func pipelineOwnerObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
	}}
}

type pipelineResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	// owner is a ListNestedBlock (MaxItems 1), so it is held as a types.List.
	Owner types.List `tfsdk:"owner"`
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
				Validators: []fwvalidator.String{
					stringvalidator.RegexMatches(
						pipelineNameRegex,
						"pipeline name must start with a lowercase letter and be 3-30 characters of lowercase letters, numbers and dashes",
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			// owner is modeled as a block to preserve the SDKv2 HCL syntax
			// (owner { ... }). The framework does not allow Computed blocks, so
			// the SDKv2 Optional+Computed behavior cannot be fully reproduced:
			// when owner is omitted, ownership still defaults to the
			// authenticated user server-side, but that default is not persisted
			// to Terraform state (a non-Computed block populated by the provider
			// would cause an "inconsistent result after apply" error).
			"owner": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []fwvalidator.String{
								pipelineOwnerIDValidator{},
							},
						},
						"type": schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []fwvalidator.String{
								pipelineOwnerTypeValidator{},
							},
						},
					},
				},
				// Restore the SDKv2 MaxItems: 1 constraint. Create reads only
				// owners[0]; without this a multi-owner config would be silently
				// truncated instead of rejected at plan time.
				Validators: []fwvalidator.List{
					listvalidator.SizeAtMost(1),
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
	opts.Owner = &struct {
		ID   string `json:"id" url:"id,key"`
		Type string `json:"type" url:"type,key"`
	}{}

	ownerConfigured := !plan.Owner.IsNull() && !plan.Owner.IsUnknown() && len(plan.Owner.Elements()) > 0
	if ownerConfigured {
		var owners []pipelineOwnerModel
		resp.Diagnostics.Append(plan.Owner.ElementsAs(ctx, &owners, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts.Owner.ID = owners[0].ID.ValueString()
		opts.Owner.Type = owners[0].Type.ValueString()
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

	resp.Diagnostics.Append(r.setModelFromPipeline(ctx, &plan, p, ownerConfigured)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// Preserve whether owner was configured: a non-Computed block must not be
	// populated by the provider when the config omits it.
	ownerConfigured := !state.Owner.IsNull() && len(state.Owner.Elements()) > 0
	resp.Diagnostics.Append(r.setModelFromPipeline(ctx, &state, p, ownerConfigured)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
	ownerConfigured := !plan.Owner.IsNull() && !plan.Owner.IsUnknown() && len(plan.Owner.Elements()) > 0
	resp.Diagnostics.Append(r.setModelFromPipeline(ctx, &plan, p, ownerConfigured)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
	// On import there is no prior config; populate owner from the API so the
	// imported state reflects the remote owner.
	resp.Diagnostics.Append(r.setModelFromPipeline(ctx, &state, p, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// setModelFromPipeline populates the model from the API response. When
// ownerConfigured is false (the owner block was omitted from config), owner is
// set to an empty block list to match the planned value: a framework block
// cannot be Computed, so the server-defaulted owner is intentionally not
// persisted to state, which would otherwise cause an "inconsistent result
// after apply" error or a perpetual diff.
func (r *pipelineResource) setModelFromPipeline(ctx context.Context, m *pipelineResourceModel, p *heroku.Pipeline, ownerConfigured bool) diag.Diagnostics {
	m.Name = types.StringValue(p.Name)

	if ownerConfigured && p.Owner != nil {
		ownerList, diags := types.ListValueFrom(ctx, pipelineOwnerObjectType(), []pipelineOwnerModel{
			{
				ID:   types.StringValue(p.Owner.ID),
				Type: types.StringValue(p.Owner.Type),
			},
		})
		m.Owner = ownerList
		return diags
	}

	emptyOwner, diags := types.ListValue(pipelineOwnerObjectType(), []attr.Value{})
	m.Owner = emptyOwner
	return diags
}
