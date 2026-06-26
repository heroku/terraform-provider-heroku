package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*pipelineConfigVarResource)(nil)
	_ resource.ResourceWithConfigure   = (*pipelineConfigVarResource)(nil)
	_ resource.ResourceWithImportState = (*pipelineConfigVarResource)(nil)
)

// NewPipelineConfigVarResource returns the framework implementation of the
// heroku_pipeline_config_var resource.
func NewPipelineConfigVarResource() resource.Resource {
	return &pipelineConfigVarResource{}
}

type pipelineConfigVarResource struct {
	config *Config
}

type pipelineConfigVarResourceModel struct {
	ID            types.String `tfsdk:"id"`
	PipelineID    types.String `tfsdk:"pipeline_id"`
	PipelineStage types.String `tfsdk:"pipeline_stage"`
	Vars          types.Map    `tfsdk:"vars"`
	SensitiveVars types.Map    `tfsdk:"sensitive_vars"`
	AllVars       types.Map    `tfsdk:"all_vars"`
}

func (r *pipelineConfigVarResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_config_var"
}

func (r *pipelineConfigVarResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 validation.IsUUID on pipeline_id.
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"pipeline_stage": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 StringInSlice([]string{"test", "review"}).
				Validators: []fwvalidator.String{
					stringvalidator.OneOf("test", "review"),
				},
			},
			"vars": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"sensitive_vars": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"all_vars": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				// Marked Sensitive so that sensitive_vars values do not leak in console/logs.
				Sensitive: true,
			},
		},
	}
}

func (r *pipelineConfigVarResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pipelineConfigVarResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pipelineConfigVarResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipelineID := plan.PipelineID.ValueString()
	pipelineStage := plan.PipelineStage.ValueString()

	vars, diags := pipelineConfigVarMapToStringInterface(ctx, plan.Vars)
	resp.Diagnostics.Append(diags...)
	sensitiveVars, diags := pipelineConfigVarMapToStringInterface(ctx, plan.SensitiveVars)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := duplicateVarsChecker(vars, sensitiveVars); err != nil {
		resp.Diagnostics.AddError("Duplicate config vars detected", err.Error())
		return
	}

	combinedVars := mergeVars(vars, sensitiveVars)

	log.Printf("[INFO] Creating pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	if err := r.callUpdatePipelineConfigVars(ctx, pipelineID, pipelineStage, nil, combinedVars); err != nil {
		resp.Diagnostics.AddError("Error creating pipeline config vars", err.Error())
		return
	}

	log.Printf("[INFO] Created pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", pipelineID, pipelineStage))

	if err := r.readIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline config vars after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelineConfigVarResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pipelineConfigVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline config vars", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pipelineConfigVarResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pipelineConfigVarResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state pipelineConfigVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID (pipeline_id and pipeline_stage are ForceNew so they won't change).
	plan.ID = state.ID

	pipelineID := plan.PipelineID.ValueString()
	pipelineStage := plan.PipelineStage.ValueString()

	oldVars, diags := pipelineConfigVarMapToStringInterface(ctx, state.Vars)
	resp.Diagnostics.Append(diags...)
	newVars, diags := pipelineConfigVarMapToStringInterface(ctx, plan.Vars)
	resp.Diagnostics.Append(diags...)
	oldSensitiveVars, diags := pipelineConfigVarMapToStringInterface(ctx, state.SensitiveVars)
	resp.Diagnostics.Append(diags...)
	newSensitiveVars, diags := pipelineConfigVarMapToStringInterface(ctx, plan.SensitiveVars)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allOldVars := mergeVars(oldVars, oldSensitiveVars)
	allNewVars := mergeVars(newVars, newSensitiveVars)

	log.Printf("[INFO] Updating pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	if err := r.callUpdatePipelineConfigVars(ctx, pipelineID, pipelineStage, allOldVars, allNewVars); err != nil {
		resp.Diagnostics.AddError("Error updating pipeline config vars", err.Error())
		return
	}

	log.Printf("[INFO] Updated pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	if err := r.readIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline config vars after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelineConfigVarResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pipelineConfigVarResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipelineID := state.PipelineID.ValueString()
	pipelineStage := state.PipelineStage.ValueString()

	vars, diags := pipelineConfigVarMapToStringInterface(ctx, state.Vars)
	resp.Diagnostics.Append(diags...)
	sensitiveVars, diags := pipelineConfigVarMapToStringInterface(ctx, state.SensitiveVars)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allVars := mergeVars(vars, sensitiveVars)

	log.Printf("[INFO] Removing pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)

	// Delete all config vars by setting them to nil.
	if err := r.callUpdatePipelineConfigVars(ctx, pipelineID, pipelineStage, allVars, nil); err != nil {
		resp.Diagnostics.AddError("Error deleting pipeline config vars", err.Error())
		return
	}

	log.Printf("[INFO] Removed pipeline [%s] stage [%s] config vars", pipelineID, pipelineStage)
}

// ImportState mirrors the SDKv2 resourceHerokuPipelineConfigVarImport, which
// always returns an error: this resource cannot be imported because its config
// vars are derived rather than uniquely addressable by a stable import ID.
func (r *pipelineConfigVarResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import not supported",
		"not possible to import this resource",
	)
}

// readIntoModel fetches the current remote state and populates the model,
// mirroring the SDKv2 resourceHerokuPipelineConfigVarRead logic exactly.
func (r *pipelineConfigVarResource) readIntoModel(ctx context.Context, m *pipelineConfigVarResourceModel) error {
	pipelineID, pipelineStage, err := parseCompositeID(m.ID.ValueString())
	if err != nil {
		return err
	}

	remotePipelineVars, err := r.config.Api.PipelineConfigVarInfoForApp(ctx, pipelineID, pipelineStage)
	if err != nil {
		return err
	}

	// Convert the remote map[string]*string to map[string]string.
	rpvFormatted := make(map[string]string)
	for key, value := range remotePipelineVars {
		rpvFormatted[key] = *value
	}

	// Extract current vars and sensitiveVars from state to vet against remote.
	currentVars, diags := pipelineConfigVarMapToStringInterface(ctx, m.Vars)
	if diags.HasError() {
		return fmt.Errorf("error reading vars from state")
	}
	currentSensitiveVars, diags := pipelineConfigVarMapToStringInterface(ctx, m.SensitiveVars)
	if diags.HasError() {
		return fmt.Errorf("error reading sensitive_vars from state")
	}

	vettedVars, vettedSensitiveVars := vetVarsForState(currentVars, currentSensitiveVars, rpvFormatted)

	log.Printf("[DEBUG] pipeline config vars to be set in state: *%#v", vettedVars)
	log.Printf("[DEBUG] pipeline sensitive config vars to be set in state: *%#v", vettedSensitiveVars)

	m.PipelineID = types.StringValue(pipelineID)
	m.PipelineStage = types.StringValue(pipelineStage)

	varsMap, diags := pipelineConfigVarStringMapToTypesMap(ctx, vettedVars)
	if diags.HasError() {
		return fmt.Errorf("error converting vars to framework map")
	}
	m.Vars = varsMap

	sensitiveVarsMap, diags := pipelineConfigVarStringMapToTypesMap(ctx, vettedSensitiveVars)
	if diags.HasError() {
		return fmt.Errorf("error converting sensitive_vars to framework map")
	}
	m.SensitiveVars = sensitiveVarsMap

	allVarsMap, diags := pipelineConfigVarStringMapToTypesMap(ctx, rpvFormatted)
	if diags.HasError() {
		return fmt.Errorf("error converting all_vars to framework map")
	}
	m.AllVars = allVarsMap

	return nil
}

// callUpdatePipelineConfigVars calls the Heroku API to update pipeline config
// vars, mirroring the SDKv2 updatePipelineConfigVars helper. The method name
// is prefixed to avoid a symbol collision with the package-level function of
// the same name defined in resource_heroku_pipeline_config_var.go.
func (r *pipelineConfigVarResource) callUpdatePipelineConfigVars(ctx context.Context, pipelineID, pipelineStage string,
	oldVars, newVars map[string]interface{}) error {
	varsToModify := constructVars(oldVars, newVars)

	log.Printf("[INFO] Modifying pipeline [%s] stage [%s] config vars: *%#v", pipelineID, pipelineStage, varsToModify)

	if _, err := r.config.Api.PipelineConfigVarUpdate(ctx, pipelineID, pipelineStage, varsToModify); err != nil {
		return fmt.Errorf("error updating pipeline config vars: %s", err)
	}

	log.Printf("[INFO] Modified pipeline [%s] stage [%s] config vars: *%#v", pipelineID, pipelineStage, varsToModify)

	return nil
}

// pipelineConfigVarMapToStringInterface converts a types.Map (with string
// element type) to map[string]interface{} for use with the shared SDKv2-era
// helpers (mergeVars, constructVars, vetVarsForState, duplicateVarsChecker).
// Returns nil when the framework map is null or unknown, matching the SDKv2
// behaviour of getVars/getSensitiveVars returning nil.
func pipelineConfigVarMapToStringInterface(ctx context.Context, m types.Map) (map[string]interface{}, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}

	raw := make(map[string]string)
	diags := m.ElementsAs(ctx, &raw, false)
	if diags.HasError() {
		return nil, diags
	}

	result := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		result[k] = v
	}
	return result, diags
}

// pipelineConfigVarStringMapToTypesMap converts a map[string]string to a
// framework types.Map with element type types.StringType.
func pipelineConfigVarStringMapToTypesMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	return types.MapValue(types.StringType, elems)
}
