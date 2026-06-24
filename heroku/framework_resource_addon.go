package heroku

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

// customAddonNameRegex matches the original SDKv2 validateCustomAddonName regex:
// must start with a letter, then letters, digits, underscores or dashes.
var customAddonNameRegex = regexp.MustCompile(`^[a-zA-Z][A-Za-z0-9_-]+$`)

// customAddonNameValidator validates the optional custom addon name at plan
// time, mirroring the SDKv2 validateCustomAddonName ValidateFunc: length 1-256
// and the customAddonNameRegex format. Without it, invalid names fall through to
// the Heroku API, which rejects them only at apply time.
type customAddonNameValidator struct{}

func (v customAddonNameValidator) Description(_ context.Context) string {
	return "custom addon name must be 1-256 characters, start with a letter, and contain only letters, numbers, underscores and dashes"
}

func (v customAddonNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v customAddonNameValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	// Skip when the attribute is absent or not yet known: the name is then
	// computed by Heroku and there is nothing to validate.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if l := len(value); l < 1 || l > 256 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid custom addon name",
			fmt.Sprintf("%q must be between 1 and 256 characters, got %d", req.Path, l),
		)
	}
	if !customAddonNameRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid custom addon name",
			"must start with a letter and can only contain lowercase letters, numbers, and dashes",
		)
	}
}

// addonResourceLock prevents parallelism for heroku_addon since the Heroku API
// cannot handle a single application requesting multiple addons simultaneously.
var addonResourceLock sync.Mutex

var (
	_ resource.Resource                 = (*addonResource)(nil)
	_ resource.ResourceWithConfigure    = (*addonResource)(nil)
	_ resource.ResourceWithImportState  = (*addonResource)(nil)
	_ resource.ResourceWithUpgradeState = (*addonResource)(nil)
)

// NewAddonResource returns the framework implementation of the heroku_addon resource.
func NewAddonResource() resource.Resource {
	return &addonResource{}
}

type addonResource struct {
	config *Config
}

type addonResourceModel struct {
	ID              types.String `tfsdk:"id"`
	AppID           types.String `tfsdk:"app_id"`
	Plan            types.String `tfsdk:"plan"`
	Name            types.String `tfsdk:"name"`
	Config          types.Map    `tfsdk:"config"`
	ProviderID      types.String `tfsdk:"provider_id"`
	ConfigVars      types.List   `tfsdk:"config_vars"`
	ConfigVarValues types.Map    `tfsdk:"config_var_values"`
}

// addonConfigRequiresReplace is a planmodifier.Map that marks the resource for
// replacement whenever the config attribute changes, mirroring ForceNew in SDKv2.
type addonConfigRequiresReplace struct{}

func (m addonConfigRequiresReplace) Description(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m addonConfigRequiresReplace) MarkdownDescription(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m addonConfigRequiresReplace) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.RequiresReplace = true
}

func (r *addonResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_addon"
}

func (r *addonResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 3,
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
			"plan": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []fwvalidator.String{
					customAddonNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					// Retain the API-generated name when absent from config;
					// otherwise it plans as unknown on every update.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// config is ForceNew — any change requires recreation.
			"config": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Map{
					addonConfigRequiresReplace{},
				},
			},
			"provider_id": schema.StringAttribute{
				Computed: true,
			},
			"config_vars": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"config_var_values": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *addonResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *addonResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	addonResourceLock.Lock()
	defer addonResourceLock.Unlock()

	var plan addonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.config.Api

	appID := plan.AppID.ValueString()
	app, err := client.AppInfo(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading app for addon", fmt.Sprintf("Error reading app for addon: %s", err))
		return
	}

	opts := heroku.AddOnCreateOpts{
		Plan:    plan.Plan.ValueString(),
		Confirm: &app.Name,
	}

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		configMap := make(map[string]string)
		resp.Diagnostics.Append(plan.Config.ElementsAs(ctx, &configMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(configMap) > 0 {
			opts.Config = configMap
		}
	}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() && plan.Name.ValueString() != "" {
		n := plan.Name.ValueString()
		opts.Name = &n
	}

	log.Printf("[DEBUG] Addon create configuration: %#v, %#v", appID, opts)
	addon, err := client.AddOnCreate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating addon",
			fmt.Sprintf("Error creating addon %s for app %s (%s): %s", plan.Plan.ValueString(), app.Name, appID, err),
		)
		return
	}

	// Wait for the addon to be provisioned using a ctx-aware polling loop,
	// using config.AddonCreateTimeout as the deadline (mirrors StateChangeConf
	// with Timeout: time.Duration(config.AddonCreateTimeout) * time.Minute).
	log.Printf("[DEBUG] Waiting for Addon (%s) to be provisioned", addon.ID)
	deadline := time.Now().Add(time.Duration(r.config.AddonCreateTimeout) * time.Minute)
	for {
		current, pollErr := client.AddOnInfoByApp(ctx, appID, addon.ID)
		if pollErr != nil {
			resp.Diagnostics.AddError(
				"Error polling addon state",
				fmt.Sprintf("Error waiting for Addon (%s) to be provisioned: %s", addon.ID, pollErr),
			)
			return
		}
		if current.State == "provisioned" {
			addon = current
			break
		}
		if current.State != "provisioning" {
			resp.Diagnostics.AddError(
				"Addon provisioning failed",
				fmt.Sprintf("Addon (%s) entered unexpected state %q while waiting for provisioning", addon.ID, current.State),
			)
			return
		}
		if time.Now().After(deadline) {
			resp.Diagnostics.AddError(
				"Timeout waiting for addon",
				fmt.Sprintf("Timeout waiting for Addon (%s) to be provisioned after %d minutes", addon.ID, r.config.AddonCreateTimeout),
			)
			return
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Context cancelled while waiting for addon",
				fmt.Sprintf("Context cancelled while waiting for Addon (%s) to be provisioned", addon.ID),
			)
			return
		case <-time.After(10 * time.Second):
		}
	}
	log.Printf("[INFO] Addon provisioned: %s", addon.ID)

	plan.ID = types.StringValue(addon.ID)
	log.Printf("[INFO] Addon ID: %s", addon.ID)

	// If SetAddonConfigVarsInState is enabled, retry until all config vars are
	// visible in the app's config vars (mirrors resource.Retry with
	// d.Timeout(schema.TimeoutCreate)).
	if r.config.SetAddonConfigVarsInState {
		cvDeadline := time.Now().Add(time.Duration(r.config.AddonCreateTimeout) * time.Minute)
		for {
			configVarValues, cvErr := retrieveAddonConfigVars(ctx, client, addon.App.ID, addon.ConfigVars)
			if cvErr != nil {
				resp.Diagnostics.AddError("Error retrieving addon config vars", cvErr.Error())
				return
			}
			if len(configVarValues) == len(addon.ConfigVars) {
				log.Printf("[INFO] Addon config vars are set: %v", addon.ConfigVars)
				break
			}
			if time.Now().After(cvDeadline) {
				resp.Diagnostics.AddError(
					"Timeout waiting for addon config vars",
					fmt.Sprintf("Got %d add-on config vars from the app, but expected %d", len(configVarValues), len(addon.ConfigVars)),
				)
				return
			}
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Context cancelled while waiting for addon config vars", ctx.Err().Error())
				return
			case <-time.After(5 * time.Second):
			}
		}
	}

	if err := r.readAddonIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading addon after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addonResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state addonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readAddonIntoModel(ctx, &state); err != nil {
		if isAddonNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading addon", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *addonResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan addonResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state addonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.config.Api
	opts := heroku.AddOnUpdateOpts{}

	if !plan.Plan.Equal(state.Plan) {
		opts.Plan = plan.Plan.ValueString()
	}

	if !plan.Name.Equal(state.Name) {
		n := plan.Name.ValueString()
		opts.Name = &n
	}

	ad, err := client.AddOnUpdate(ctx, state.AppID.ValueString(), state.ID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating addon", err.Error())
		return
	}

	// Store the new addon ID — it may change on plan upgrade.
	plan.ID = types.StringValue(ad.ID)

	if err := r.readAddonIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading addon after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addonResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state addonResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting Addon: %s", state.ID.ValueString())

	_, err := r.config.Api.AddOnDelete(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil && !isAddonNotFound(err) {
		resp.Diagnostics.AddError("Error deleting addon", fmt.Sprintf("Error deleting addon: %s", err))
		return
	}
}

func (r *addonResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// addonResourceModelPriorV3 mirrors the pre-v3 SDKv2 schema, which used the
// fuzzy "app" identifier instead of "app_id". It is the prior model decoded by
// the state upgraders before the value is resolved to a UUID.
type addonResourceModelPriorV3 struct {
	ID              types.String `tfsdk:"id"`
	App             types.String `tfsdk:"app"`
	Plan            types.String `tfsdk:"plan"`
	Name            types.String `tfsdk:"name"`
	Config          types.Map    `tfsdk:"config"`
	ProviderID      types.String `tfsdk:"provider_id"`
	ConfigVars      types.List   `tfsdk:"config_vars"`
	ConfigVarValues types.Map    `tfsdk:"config_var_values"`
}

// UpgradeState ports the SDKv2 resourceHerokuAddonMigrate behavior to the
// framework. The shipped SDKv2 provider declared SchemaVersion 3 with a legacy
// MigrateState (flatmap) path for v0->v1 (addon name->UUID), v1->v2 (config
// TypeList->TypeSet) and v2->v3 (app->app_id).
//
// Of these, only the v2->v3 app->app_id resolution is faithfully reproducible
// against JSON state, so it is implemented as a real upgrader. The v0->v1 and
// v1->v2 migrations operated on pre-0.12 flatmap state that the framework never
// receives as JSON; they are registered as log-and-passthrough upgraders that
// still resolve app->app_id (best effort) and warn that the flatmap-specific
// transforms cannot be applied.
func (r *addonResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// priorSchema mirrors the pre-v3 schema: "app" (fuzzy id) instead of
	// "app_id". Computed attributes decode as null when absent from older state.
	priorSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true},
			"app":  schema.StringAttribute{Required: true},
			"plan": schema.StringAttribute{Required: true},
			"name": schema.StringAttribute{Optional: true, Computed: true},
			"config": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"provider_id": schema.StringAttribute{Computed: true},
			"config_vars": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"config_var_values": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}

	// upgradeToV3 decodes the prior state, resolves app->app_id, and writes the
	// current (v3) model. fromVersion drives the warning emitted for the
	// unreachable flatmap-era versions.
	upgradeToV3 := func(fromVersion int) func(context.Context, resource.UpgradeStateRequest, *resource.UpgradeStateResponse) {
		return func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			if fromVersion < 2 {
				log.Printf("[WARN] heroku_addon state v%d detected. The v%d migration was a pre-0.12 flatmap MigrateState transform (addon-id and config-format) that cannot be reproduced for JSON state; resolving app_id only.", fromVersion, fromVersion)
			}

			var prior addonResourceModelPriorV3
			resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
			if resp.Diagnostics.HasError() {
				return
			}

			appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
			if err != nil {
				resp.Diagnostics.AddError("Error upgrading heroku_addon state", err.Error())
				return
			}

			upgraded := addonResourceModel{
				ID:              prior.ID,
				AppID:           types.StringValue(appID),
				Plan:            prior.Plan,
				Name:            prior.Name,
				Config:          prior.Config,
				ProviderID:      prior.ProviderID,
				ConfigVars:      prior.ConfigVars,
				ConfigVarValues: prior.ConfigVarValues,
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
		}
	}

	return map[int64]resource.StateUpgrader{
		0: {PriorSchema: &priorSchema, StateUpgrader: upgradeToV3(0)},
		1: {PriorSchema: &priorSchema, StateUpgrader: upgradeToV3(1)},
		2: {PriorSchema: &priorSchema, StateUpgrader: upgradeToV3(2)},
	}
}

// readAddonIntoModel fetches the addon by ID and populates the model, mirroring
// the SDKv2 resourceHerokuAddonRead logic faithfully.
func (r *addonResource) readAddonIntoModel(ctx context.Context, m *addonResourceModel) error {
	client := r.config.Api

	addon, err := client.AddOnInfo(ctx, m.ID.ValueString())
	if err != nil {
		return fmt.Errorf("Error retrieving addon: %w", err)
	}

	// Determine the plan. If the stored plan has no colon-separated tier, strip
	// the tier from the API response so that plan changes are not flagged on
	// every refresh (mirrors SDKv2 resourceHerokuAddonRead logic).
	plan := addon.Plan.Name
	if v := m.Plan.ValueString(); v != "" {
		if idx := strings.IndexRune(v, ':'); idx == -1 {
			if idx2 := strings.IndexRune(plan, ':'); idx2 > -1 {
				plan = plan[:idx2]
			}
		}
	}

	m.Name = types.StringValue(addon.Name)
	m.AppID = types.StringValue(addon.App.ID)
	m.Plan = types.StringValue(plan)
	m.ProviderID = types.StringValue(addon.ProviderID)

	configVarsVal, diags := types.ListValueFrom(ctx, types.StringType, addon.ConfigVars)
	if diags.HasError() {
		return fmt.Errorf("error building config_vars list: %v", diags)
	}
	m.ConfigVars = configVarsVal

	// Always initialise config_var_values to an empty map; override below if
	// SetAddonConfigVarsInState is enabled.
	emptyMap, diags := types.MapValue(types.StringType, map[string]attr.Value{})
	if diags.HasError() {
		return fmt.Errorf("error building empty config_var_values map: %v", diags)
	}
	m.ConfigVarValues = emptyMap

	if r.config.SetAddonConfigVarsInState {
		configVarValues, err := retrieveAddonConfigVars(ctx, client, addon.App.ID, addon.ConfigVars)
		if err != nil {
			return err
		}
		cvvMap, diags := types.MapValueFrom(ctx, types.StringType, configVarValues)
		if diags.HasError() {
			return fmt.Errorf("error building config_var_values map: %v", diags)
		}
		m.ConfigVarValues = cvvMap
	}

	return nil
}

// retrieveAddonConfigVars fetches the app's config vars and returns only those
// whose keys appear in varNames, filtering out null values. Mirrors the SDKv2
// retrieveSpecificConfigVars function but accepts a context for framework use.
func retrieveAddonConfigVars(ctx context.Context, client *heroku.Service, appID string, varNames []string) (map[string]string, error) {
	vars, err := client.ConfigVarInfoForApp(ctx, appID)
	if err != nil {
		return nil, err
	}

	nonNullVars := map[string]string{}
	for k, v := range vars {
		if SliceContainsString(varNames, k) && v != nil {
			nonNullVars[k] = *v
		}
	}

	return nonNullVars, nil
}

// isAddonNotFound returns true when err represents a Heroku 404/not_found. The
// heroku-go client returns heroku.Error (a value type); callers must wrap it
// with fmt.Errorf("...: %w") so errors.As can walk the chain.
func isAddonNotFound(err error) bool {
	if err == nil {
		return false
	}
	var herr heroku.Error
	if errors.As(err, &herr) {
		return herr.ID == "not_found"
	}
	return false
}
