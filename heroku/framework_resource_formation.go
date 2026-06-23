package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*formationResource)(nil)
	_ resource.ResourceWithConfigure    = (*formationResource)(nil)
	_ resource.ResourceWithImportState  = (*formationResource)(nil)
	_ resource.ResourceWithUpgradeState = (*formationResource)(nil)
)

// NewFormationResource returns the framework implementation of the
// heroku_formation resource.
func NewFormationResource() resource.Resource {
	return &formationResource{}
}

type formationResource struct {
	config *Config
}

// formationResourceModel is the model for the current schema (version 1).
type formationResourceModel struct {
	ID       types.String `tfsdk:"id"`
	AppID    types.String `tfsdk:"app_id"`
	Type     types.String `tfsdk:"type"`
	Quantity types.Int64  `tfsdk:"quantity"`
	Size     types.String `tfsdk:"size"`
}

// formationResourceModelV0 mirrors resourceHerokuFormationV0(): uses "app" instead of "app_id".
type formationResourceModelV0 struct {
	ID       types.String `tfsdk:"id"`
	App      types.String `tfsdk:"app"`
	Type     types.String `tfsdk:"type"`
	Quantity types.Int64  `tfsdk:"quantity"`
	Size     types.String `tfsdk:"size"`
}

// caseInsensitiveStringPlanModifier is a planmodifier.String that suppresses
// plan differences caused solely by case changes (e.g. dyno size names differ
// between Cedar and Fir API responses). It mirrors the SDKv2 suppressCaseDiff
// DiffSuppressFunc.
type caseInsensitiveStringPlanModifier struct{}

func (m caseInsensitiveStringPlanModifier) Description(_ context.Context) string {
	return "Suppress case-only differences in string values."
}

func (m caseInsensitiveStringPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Suppress case-only differences in string values."
}

func (m caseInsensitiveStringPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the state and plan values differ only in case, keep the state value so
	// no diff is presented to the user.
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() &&
		!req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() {
		if strings.EqualFold(req.StateValue.ValueString(), req.PlanValue.ValueString()) {
			resp.PlanValue = req.StateValue
		}
	}
}

func (r *formationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_formation"
}

func (r *formationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			},
			"type": schema.StringAttribute{
				Required: true,
			},
			"quantity": schema.Int64Attribute{
				Required: true,
			},
			"size": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					caseInsensitiveStringPlanModifier{},
				},
			},
		},
	}
}

func (r *formationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create calls FormationUpdate (there is no dedicated CREATE endpoint for
// formations; the SDKv2 resourceHerokuFormationCreate uses FormationUpdate too).
// It also validates the app exists first, mirroring the SDKv2 doesHerokuAppExist check.
func (r *formationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan formationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()

	// Validate the app exists, mirroring the SDKv2 doesHerokuAppExist call.
	if _, err := r.config.Api.AppInfo(ctx, appID); err != nil {
		resp.Diagnostics.AddError("[ERROR] Your app does not exist", err.Error())
		return
	}

	opts := r.buildUpdateOpts(plan)

	log.Printf("[DEBUG] Updating %s formation...", appID)
	f, err := r.config.Api.FormationUpdate(ctx, appID, plan.Type.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating/updating formation", err.Error())
		return
	}

	plan.ID = types.StringValue(f.ID)
	log.Printf("[INFO] Formation ID: %s", f.ID)

	if err := r.readFormationIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading formation after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *formationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state formationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readFormationIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading formation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *formationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan formationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	var state formationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	appID := plan.AppID.ValueString()

	// Validate the app exists, mirroring the SDKv2 doesHerokuAppExist call.
	if _, err := r.config.Api.AppInfo(ctx, appID); err != nil {
		resp.Diagnostics.AddError("[ERROR] Your app does not exist", err.Error())
		return
	}

	opts := r.buildUpdateOpts(plan)

	log.Printf("[DEBUG] Updating Heroku formation...")
	updatedFormation, err := r.config.Api.FormationUpdate(ctx, appID, plan.Type.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating formation", err.Error())
		return
	}

	plan.ID = types.StringValue(updatedFormation.ID)

	if err := r.readFormationIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading formation after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op: there is no DELETE endpoint for the formation resource.
// The resource is simply removed from state, mirroring resourceHerokuFormationDelete.
func (r *formationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	log.Printf("[INFO] There is no DELETE for formation resource so this is a no-op. Resource will be removed from state.")
}

func (r *formationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, formationType, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing formation", err.Error())
		return
	}

	f, err := r.config.Api.FormationInfo(ctx, app, formationType)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving formation during import", err.Error())
		return
	}

	state := formationResourceModel{
		ID:       types.StringValue(f.ID),
		AppID:    types.StringValue(f.App.ID),
		Type:     types.StringValue(f.Type),
		Quantity: types.Int64Value(int64(f.Quantity)),
		Size:     types.StringValue(f.Size),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// UpgradeState implements resource.ResourceWithUpgradeState.
// The SDKv2 resource had SchemaVersion 1 with a single v0→v1 upgrader that
// replaced the "app" attribute (name or UUID) with "app_id" (UUID), mirroring
// upgradeAppToAppID registered against resourceHerokuFormationV0().
func (r *formationResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuFormationV0(): "app" + "type" + "quantity" + "size".
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
			},
			"quantity": schema.Int64Attribute{
				Required: true,
			},
			"size": schema.StringAttribute{
				Required: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior formationResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading formation state", err.Error())
					return
				}

				upgraded := formationResourceModel{
					ID:       prior.ID,
					AppID:    types.StringValue(appID),
					Type:     prior.Type,
					Quantity: prior.Quantity,
					Size:     prior.Size,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// buildUpdateOpts constructs a FormationUpdateOpts from the model, mirroring
// the size/quantity population in both resourceHerokuFormationCreate and
// resourceHerokuFormationUpdate.
func (r *formationResource) buildUpdateOpts(m formationResourceModel) heroku.FormationUpdateOpts {
	opts := heroku.FormationUpdateOpts{}

	sizeVal := m.Size.ValueString()
	if sizeVal != "" {
		log.Printf("[DEBUG] Size: %s", sizeVal)
		opts.DynoSize = &struct {
			ID   *string `json:"id,omitempty" url:"id,omitempty,key"`
			Name *string `json:"name,omitempty" url:"name,omitempty,key"`
		}{
			Name: &sizeVal,
		}
	}

	quantity := int(m.Quantity.ValueInt64())
	log.Printf("[DEBUG] Quantity: %v", quantity)
	opts.Quantity = &quantity

	return opts
}

// readFormationIntoModel fetches current formation state from the Heroku API
// and populates the model. The formation is looked up by its ID (UUID) within
// the app, mirroring the SDKv2 formation.GetInfo logic.
func (r *formationResource) readFormationIntoModel(ctx context.Context, m *formationResourceModel) error {
	appID := m.AppID.ValueString()
	formationID := m.ID.ValueString()

	log.Printf("[INFO] Reading formation %s for app %s", formationID, appID)

	f, err := r.config.Api.FormationInfo(ctx, appID, formationID)
	if err != nil {
		return fmt.Errorf("error retrieving formation: %s", err)
	}

	m.AppID = types.StringValue(f.App.ID)
	m.Type = types.StringValue(f.Type)
	m.Quantity = types.Int64Value(int64(f.Quantity))
	m.Size = types.StringValue(f.Size)

	return nil
}
