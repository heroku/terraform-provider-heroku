package heroku

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*configResource)(nil)
	_ resource.ResourceWithConfigure   = (*configResource)(nil)
	_ resource.ResourceWithImportState = (*configResource)(nil)
)

// NewConfigResource returns the framework implementation of the heroku_config resource.
func NewConfigResource() resource.Resource {
	return &configResource{}
}

type configResource struct {
	config *Config
}

type configResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Vars          types.Map    `tfsdk:"vars"`
	SensitiveVars types.Map    `tfsdk:"sensitive_vars"`
}

func (r *configResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (r *configResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

func (r *configResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *configResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan configResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vars := mapFromState(plan.Vars)
	sensitiveVars := mapFromState(plan.SensitiveVars)

	log.Printf("[DEBUG] vars: %v", vars)
	log.Printf("[DEBUG] sensitive vars: %v", sensitiveVars)

	if err := duplicateVarsCheckerString(vars, sensitiveVars); err != nil {
		resp.Diagnostics.AddError("Duplicate config vars detected", err.Error())
		return
	}

	// Set the ID to config-<epoch> for uniqueness, matching the SDKv2 behaviour.
	epochTimeString := strconv.FormatInt(time.Now().Unix(), 10)
	plan.ID = types.StringValue(fmt.Sprintf("config-%s", epochTimeString))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state configResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// heroku_config has no remote resource; state is the source of truth.
	// Re-set exactly what is already in state (mirrors SDKv2 resourceHerokuConfigRead).
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *configResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan configResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vars := mapFromState(plan.Vars)
	sensitiveVars := mapFromState(plan.SensitiveVars)

	log.Printf("[DEBUG] vars: %v", vars)
	log.Printf("[DEBUG] sensitive vars: %v", sensitiveVars)

	if err := duplicateVarsCheckerString(vars, sensitiveVars); err != nil {
		resp.Diagnostics.AddError("Duplicate config vars detected", err.Error())
		return
	}

	// Preserve the existing ID from state during updates.
	var state configResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *configResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	log.Printf("[INFO] There is no DELETE for config resource since no data is stored in Heroku. " +
		"Resource will be removed from state.")
	// Framework removes the resource from state automatically after Delete returns.
}

// ImportState mirrors the SDKv2 importer, which explicitly rejects import because
// heroku_config has no remote resource to import from.
func (r *configResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import not supported",
		"It is not possible to import heroku_config since there are no remote resources associated with heroku_config.",
	)
}

// mapFromState converts a types.Map (of string elements) to map[string]string.
// Returns an empty map if the value is null or unknown.
func mapFromState(m types.Map) map[string]string {
	result := make(map[string]string)
	if m.IsNull() || m.IsUnknown() {
		return result
	}
	for k, v := range m.Elements() {
		if sv, ok := v.(types.String); ok && !sv.IsNull() && !sv.IsUnknown() {
			result[k] = sv.ValueString()
		}
	}
	return result
}

// duplicateVarsCheckerString is the framework-layer analogue of duplicateVarsChecker.
// It works on map[string]string instead of map[string]interface{} so it does not
// depend on the SDKv2 helper defined in resource_heroku_config.go.
func duplicateVarsCheckerString(vars, sensitiveVars map[string]string) error {
	var dupes []string
	for k := range sensitiveVars {
		if _, ok := vars[k]; ok {
			dupes = append(dupes, k)
		}
	}
	log.Printf("[INFO] List of Duplicate config vars (if any) %v", dupes)
	if len(dupes) > 0 {
		return fmt.Errorf("[ERROR] Detected duplicate config vars: %v", dupes)
	}
	return nil
}
