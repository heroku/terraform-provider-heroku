package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource              = (*appConfigAssociationResource)(nil)
	_ resource.ResourceWithConfigure = (*appConfigAssociationResource)(nil)
)

// NewAppConfigAssociationResource returns the framework implementation of the
// heroku_app_config_association resource.
func NewAppConfigAssociationResource() resource.Resource {
	return &appConfigAssociationResource{}
}

type appConfigAssociationResource struct {
	config *Config
}

type appConfigAssociationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	AppID         types.String `tfsdk:"app_id"`
	Vars          types.Map    `tfsdk:"vars"`
	SensitiveVars types.Map    `tfsdk:"sensitive_vars"`
}

func (r *appConfigAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_config_association"
}

func (r *appConfigAssociationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Required: true,
				// TODO: port validation.IsUUID
			},
			"vars": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"sensitive_vars": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *appConfigAssociationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *appConfigAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appConfigAssociationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()
	vars := mapFromState(plan.Vars)
	sensitiveVars := mapFromState(plan.SensitiveVars)

	// Check for duplicates between vars & sensitive_vars.
	if err := duplicateVarsCheckerString(vars, sensitiveVars); err != nil {
		resp.Diagnostics.AddError("Duplicate config vars", err.Error())
		return
	}

	combinedVars := mergeStringMaps(vars, sensitiveVars)

	if err := r.applyConfigVars(ctx, appID, nil, combinedVars); err != nil {
		resp.Diagnostics.AddError("Error setting config vars", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("config:%s", appID))

	if err := r.readIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading config vars after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appConfigAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appConfigAssociationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading config vars", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *appConfigAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appConfigAssociationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state appConfigAssociationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve existing ID.
	plan.ID = state.ID
	appID := plan.AppID.ValueString()

	allOldVars := mergeStringMaps(mapFromState(state.Vars), mapFromState(state.SensitiveVars))
	allNewVars := mergeStringMaps(mapFromState(plan.Vars), mapFromState(plan.SensitiveVars))

	if err := r.applyConfigVars(ctx, appID, allOldVars, allNewVars); err != nil {
		resp.Diagnostics.AddError("Error updating config vars", err.Error())
		return
	}

	if err := r.readIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading config vars after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appConfigAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appConfigAssociationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := state.AppID.ValueString()
	allVars := mergeStringMaps(mapFromState(state.Vars), mapFromState(state.SensitiveVars))

	// Delete by patching all managed vars to nil (removing them from the app).
	if err := r.applyConfigVars(ctx, appID, allVars, nil); err != nil {
		resp.Diagnostics.AddError("Error deleting config vars", err.Error())
		return
	}
}

// readIntoModel fetches remote config vars and vets them against the locally
// managed vars and sensitive_vars, then populates the model — mirroring the
// SDKv2 resourceHerokuAppConfigAssociationRead logic exactly.
func (r *appConfigAssociationResource) readIntoModel(ctx context.Context, m *appConfigAssociationResourceModel) error {
	appID := m.AppID.ValueString()

	remoteVars, err := retrieveConfigVars(appID, r.config.Api)
	if err != nil {
		return err
	}

	currentVars := mapFromState(m.Vars)
	currentSensitiveVars := mapFromState(m.SensitiveVars)

	vettedVars, vettedSensitiveVars := vetVarsForState(
		stringMapToInterfaceMap(currentVars),
		stringMapToInterfaceMap(currentSensitiveVars),
		remoteVars,
	)

	varsMap, varsDiags := types.MapValueFrom(ctx, types.StringType, vettedVars)
	if varsDiags.HasError() {
		log.Printf("[WARN] Error setting app config vars")
	} else {
		m.Vars = varsMap
	}

	sensitiveMap, sensitiveDiags := types.MapValueFrom(ctx, types.StringType, vettedSensitiveVars)
	if sensitiveDiags.HasError() {
		log.Printf("[WARN] Error setting app config sensitive vars")
	} else {
		m.SensitiveVars = sensitiveMap
	}

	return nil
}

// applyConfigVars constructs and applies a config var update via the Heroku API,
// then waits for the resulting release to succeed — mirroring the updateVars
// helper in the SDKv2 file exactly.
func (r *appConfigAssociationResource) applyConfigVars(ctx context.Context, appID string, oldVars, newVars map[string]string) error {
	vars := buildConfigVarPatch(oldVars, newVars)

	log.Printf("[INFO] Updating config vars: *%#v", vars)
	if _, err := r.config.Api.ConfigVarUpdate(ctx, appID, vars); err != nil {
		return fmt.Errorf("error updating config vars: %s", err)
	}

	releases, err := r.config.Api.ReleaseList(
		ctx,
		appID,
		&heroku.ListRange{Descending: true, Field: "version", Max: 1},
	)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		return fmt.Errorf("no release found after config var update on app %s", appID)
	}

	if err := waitForReleaseSucceeded(ctx, r.config.Api, appID, releases[0].ID, 20*time.Minute); err != nil {
		return fmt.Errorf("error waiting for new release (%s) to succeed: %s", releases[0].ID, err)
	}

	return nil
}

// stringMapToInterfaceMap converts map[string]string to map[string]interface{}
// for compatibility with the shared helper functions that accept interface{} maps.
func stringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// mergeStringMaps combines two map[string]string values into a single map.
func mergeStringMaps(a, b map[string]string) map[string]string {
	combined := make(map[string]string)
	for k, v := range a {
		combined[k] = v
	}
	for k, v := range b {
		combined[k] = v
	}
	return combined
}

// buildConfigVarPatch builds the map[string]*string required by ConfigVarUpdate
// from old/new string maps: keys present only in oldVars are set to nil
// (deletion), keys in newVars are set to their new values.
func buildConfigVarPatch(oldVars, newVars map[string]string) map[string]*string {
	vars := make(map[string]*string)
	for k := range oldVars {
		vars[k] = nil
	}
	for k, v := range newVars {
		val := v
		vars[k] = &val
	}
	return vars
}
