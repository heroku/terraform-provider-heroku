package heroku

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*appResource)(nil)
	_ resource.ResourceWithConfigure    = (*appResource)(nil)
	_ resource.ResourceWithImportState  = (*appResource)(nil)
	_ resource.ResourceWithUpgradeState = (*appResource)(nil)
)

// NewAppResource returns the framework implementation of the heroku_app resource.
func NewAppResource() resource.Resource {
	return &appResource{}
}

type appResource struct {
	config *Config
}

// appOrgModel mirrors the organization/team nested block on heroku_app.
type appOrgModel struct {
	Name     types.String `tfsdk:"name"`
	Locked   types.Bool   `tfsdk:"locked"`
	Personal types.Bool   `tfsdk:"personal"`
}

// appResourceModel is the model for the current schema (version 1).
type appResourceModel struct {
	ID                  types.String  `tfsdk:"id"`
	Name                types.String  `tfsdk:"name"`
	Region              types.String  `tfsdk:"region"`
	Space               types.String  `tfsdk:"space"`
	Stack               types.String  `tfsdk:"stack"`
	Generation          types.String  `tfsdk:"generation"`
	InternalRouting     types.Bool    `tfsdk:"internal_routing"`
	Buildpacks          types.List    `tfsdk:"buildpacks"`
	ConfigVars          types.Map     `tfsdk:"config_vars"`
	SensitiveConfigVars types.Map     `tfsdk:"sensitive_config_vars"`
	AllConfigVars       types.Map     `tfsdk:"all_config_vars"`
	GitURL              types.String  `tfsdk:"git_url"`
	WebURL              types.String  `tfsdk:"web_url"`
	Acm                 types.Bool    `tfsdk:"acm"`
	HerokuHostname      types.String  `tfsdk:"heroku_hostname"`
	Organization        []appOrgModel `tfsdk:"organization"`
	UUID                types.String  `tfsdk:"uuid"`
}

// appResourceModelV0 is the model for the prior schema (version 0), which
// lacked the "generation" attribute.
type appResourceModelV0 struct {
	ID                  types.String  `tfsdk:"id"`
	Name                types.String  `tfsdk:"name"`
	Region              types.String  `tfsdk:"region"`
	Space               types.String  `tfsdk:"space"`
	Stack               types.String  `tfsdk:"stack"`
	InternalRouting     types.Bool    `tfsdk:"internal_routing"`
	Buildpacks          types.List    `tfsdk:"buildpacks"`
	ConfigVars          types.Map     `tfsdk:"config_vars"`
	SensitiveConfigVars types.Map     `tfsdk:"sensitive_config_vars"`
	AllConfigVars       types.Map     `tfsdk:"all_config_vars"`
	GitURL              types.String  `tfsdk:"git_url"`
	WebURL              types.String  `tfsdk:"web_url"`
	Acm                 types.Bool    `tfsdk:"acm"`
	HerokuHostname      types.String  `tfsdk:"heroku_hostname"`
	Organization        []appOrgModel `tfsdk:"organization"`
	UUID                types.String  `tfsdk:"uuid"`
}

// -----------------------------------------------------------------------------
// Inline plan modifiers expressing the SDKv2 ForceNew-on-change semantics. The
// stock boolplanmodifier.RequiresReplace replaces whenever the value differs
// from prior state, but these variants additionally skip the create case
// (null prior state), matching the original resource's behaviour.
// -----------------------------------------------------------------------------

// appBoolRequiresReplace is a planmodifier.Bool that marks the resource for
// replacement when the value changes from a known state. This mirrors the
// SDKv2 ForceNew behaviour for bool attributes.
type appBoolRequiresReplace struct{}

func (m appBoolRequiresReplace) Description(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m appBoolRequiresReplace) MarkdownDescription(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m appBoolRequiresReplace) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	// Skip on create (no prior state).
	if req.StateValue.IsNull() {
		return
	}
	// Skip while either value is unknown: a comparison against an unresolved
	// value would spuriously trigger replacement. This matches the stock
	// boolplanmodifier.RequiresReplace behaviour.
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// appListRequiresReplace is a planmodifier.List that marks the resource for
// replacement when the list value changes from a known state. Used for the
// organization block (ForceNew in SDKv2).
type appListRequiresReplace struct{}

func (m appListRequiresReplace) Description(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m appListRequiresReplace) MarkdownDescription(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m appListRequiresReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// -----------------------------------------------------------------------------
// Schema
// -----------------------------------------------------------------------------

func (r *appResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *appResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"space": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"stack": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"generation": schema.StringAttribute{
				Computed:    true,
				Description: "Generation of the app platform. Determined by the space the app is deployed to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"internal_routing": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					// Carry the prior computed value forward when the attribute is
					// absent from config. Without this the value plans as unknown on
					// every update, which both produces a perpetual diff and trips
					// appBoolRequiresReplace into a spurious destroy/recreate. In
					// SDKv2 a Computed+ForceNew attribute retained its state value
					// during planning, so ForceNew never fired on unrelated updates.
					boolplanmodifier.UseStateForUnknown(),
					appBoolRequiresReplace{},
				},
			},
			"buildpacks": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"config_vars": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"sensitive_config_vars": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"all_config_vars": schema.MapAttribute{
				Computed: true,
				// Marked sensitive so that sensitive_config_vars do not leak in
				// the console/logs and to avoid unnecessary disclosure of add-on
				// secrets in logs.
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"git_url": schema.StringAttribute{
				Computed: true,
			},
			"web_url": schema.StringAttribute{
				Computed: true,
			},
			"acm": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"heroku_hostname": schema.StringAttribute{
				Computed: true,
			},
			"uuid": schema.StringAttribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			// The "organization" block mirrors the SDKv2 TypeList MaxItems:1 block.
			// ForceNew on "name" and "personal" is expressed via RequiresReplace plan
			// modifiers on those nested attributes. The block itself is ForceNew via
			// appListRequiresReplace.
			"organization": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"locked": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"personal": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Bool{
								appBoolRequiresReplace{},
							},
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					appListRequiresReplace{},
				},
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Configure
// -----------------------------------------------------------------------------

func (r *appResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// -----------------------------------------------------------------------------
// Create
// -----------------------------------------------------------------------------

func (r *appResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.config.Api

	var appID string

	if len(plan.Organization) > 0 {
		// ---- team/org app path (mirrors resourceHerokuTeamAppCreate) ----
		if len(plan.Organization) > 1 {
			resp.Diagnostics.AddError("Error Creating Heroku App", "Only 1 Heroku Team (organization) is permitted")
			return
		}
		org := plan.Organization[0]

		opts := heroku.TeamAppCreateOpts{}

		teamName := org.Name.ValueString()
		log.Printf("[DEBUG] Organization name: %s", teamName)
		opts.Team = &teamName

		personal := org.Personal.ValueBool()
		log.Printf("[DEBUG] Organization Personal: %t", personal)
		opts.Personal = &personal

		locked := org.Locked.ValueBool()
		log.Printf("[DEBUG] Organization locked: %t", locked)
		opts.Locked = &locked

		appName := plan.Name.ValueString()
		log.Printf("[DEBUG] App name: %s", appName)
		opts.Name = &appName

		if !plan.Region.IsNull() && !plan.Region.IsUnknown() {
			region := plan.Region.ValueString()
			log.Printf("[DEBUG] App region: %s", region)
			opts.Region = &region
		}
		if !plan.Space.IsNull() && !plan.Space.IsUnknown() {
			space := plan.Space.ValueString()
			log.Printf("[DEBUG] App space: %s", space)
			opts.Space = &space
		}
		if !plan.Stack.IsNull() && !plan.Stack.IsUnknown() {
			stack := plan.Stack.ValueString()
			log.Printf("[DEBUG] App stack: %s", stack)
			opts.Stack = &stack
		}
		if !plan.InternalRouting.IsNull() && !plan.InternalRouting.IsUnknown() {
			ir := plan.InternalRouting.ValueBool()
			log.Printf("[DEBUG] App internal routing: %v", ir)
			opts.InternalRouting = &ir
		}

		log.Printf("[DEBUG] Creating Heroku team app...")
		a, err := client.TeamAppCreate(ctx, opts)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Heroku team app", err.Error())
			return
		}
		appID = a.ID
		log.Printf("[INFO] Team App ID: %s", appID)

	} else {
		// ---- personal app path (mirrors resourceHerokuAppCreate) ----
		opts := heroku.AppCreateOpts{}

		if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
			name := plan.Name.ValueString()
			log.Printf("[DEBUG] App name: %s", name)
			opts.Name = &name
		}
		if !plan.Region.IsNull() && !plan.Region.IsUnknown() {
			region := plan.Region.ValueString()
			log.Printf("[DEBUG] App region: %s", region)
			opts.Region = &region
		}
		if !plan.Stack.IsNull() && !plan.Stack.IsUnknown() {
			stack := plan.Stack.ValueString()
			log.Printf("[DEBUG] App stack: %s", stack)
			opts.Stack = &stack
		}

		log.Printf("[DEBUG] Creating Heroku app...")
		a, err := client.AppCreate(ctx, opts)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Heroku app", err.Error())
			return
		}
		appID = a.ID
		log.Printf("[INFO] App ID: %s", appID)
	}

	// ---- post-create tasks (mirrors performAppPostCreateTasks) ----

	// Check for duplicate config vars.
	configVars := mapFromState(plan.ConfigVars)
	sensitiveConfigVars := mapFromState(plan.SensitiveConfigVars)
	if err := duplicateVarsCheckerString(configVars, sensitiveConfigVars); err != nil {
		resp.Diagnostics.AddError("Duplicate config vars detected", err.Error())
		return
	}

	// Apply config vars (combine both maps then call updateConfigVars with nil old).
	allConfigVars := mergeStringMaps(configVars, sensitiveConfigVars)
	if len(allConfigVars) > 0 {
		allConfigVarsI := stringMapToInterfaceMap(allConfigVars)
		if err := updateConfigVars(appID, client, nil, allConfigVarsI); err != nil {
			resp.Diagnostics.AddError("Error setting config vars after app create", err.Error())
			return
		}
	}

	// Apply buildpacks.
	buildpackStrings := appBuildpacksToStringSlice(plan.Buildpacks)
	if len(buildpackStrings) > 0 {
		bpInterfaces := make([]interface{}, len(buildpackStrings))
		for i, s := range buildpackStrings {
			bpInterfaces[i] = s
		}
		if err := updateBuildpacks(appID, client, bpInterfaces); err != nil {
			resp.Diagnostics.AddError("Error setting buildpacks after app create", err.Error())
			return
		}
	}

	// Apply ACM. Only act when enabling: a freshly created app has ACM disabled,
	// so attempting to disable it errors ("Your app does not have ACM enabled").
	// This mirrors the SDKv2 create path, which gated on d.GetOk("acm") (false
	// for the zero value) and therefore only ever enabled ACM on create.
	if !plan.Acm.IsNull() && !plan.Acm.IsUnknown() && plan.Acm.ValueBool() {
		if len(plan.Organization) == 0 {
			log.Printf("You ask me to enable ACM for a non-organization app. This will most likely fail, " +
				"due to the Heroku constraints (the app has to be scaled to Standard-1X - state of 28.01.2018)")
		}
		if err := updateAcm(appID, client, true); err != nil {
			resp.Diagnostics.AddError("Error setting ACM after app create", err.Error())
			return
		}
	}

	// Apply the post-app-create delay.
	time.Sleep(time.Duration(r.config.PostAppCreateDelay) * time.Second)

	// Read back the full state.
	plan.ID = types.StringValue(appID)
	if err := r.readIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading app after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// -----------------------------------------------------------------------------
// Read
// -----------------------------------------------------------------------------

func (r *appResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readIntoModel(ctx, &state); err != nil {
		// If the app no longer exists, remove from state.
		if isAppNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Heroku app", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

func (r *appResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state appResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the ID from state.
	plan.ID = state.ID
	appID := state.ID.ValueString()
	client := r.config.Api

	// ---- App-level attributes (name / stack) ----
	opts := heroku.AppUpdateOpts{}
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		opts.Name = &name
	}
	if !plan.Stack.Equal(state.Stack) {
		stack := plan.Stack.ValueString()
		opts.BuildStack = &stack
	}

	updatedApp, err := client.AppUpdate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Heroku app", err.Error())
		return
	}
	// Sync the name back; the API may normalise it.
	plan.Name = types.StringValue(updatedApp.Name)

	// ---- Buildpacks ----
	if !buildpackListsEqual(plan.Buildpacks, state.Buildpacks) {
		bpStrings := appBuildpacksToStringSlice(plan.Buildpacks)
		bpInterfaces := make([]interface{}, len(bpStrings))
		for i, s := range bpStrings {
			bpInterfaces[i] = s
		}
		if err := updateBuildpacks(appID, client, bpInterfaces); err != nil {
			resp.Diagnostics.AddError("Error updating buildpacks", err.Error())
			return
		}
	}

	// ---- Config vars ----
	// Check for duplicates in the new plan first.
	planConfigVars := mapFromState(plan.ConfigVars)
	planSensitiveConfigVars := mapFromState(plan.SensitiveConfigVars)
	if err := duplicateVarsCheckerString(planConfigVars, planSensitiveConfigVars); err != nil {
		resp.Diagnostics.AddError("Duplicate config vars detected", err.Error())
		return
	}

	stateConfigVars := mapFromState(state.ConfigVars)
	stateSensitiveConfigVars := mapFromState(state.SensitiveConfigVars)

	allOldVars := stringMapToInterfaceMap(mergeStringMaps(stateConfigVars, stateSensitiveConfigVars))
	allNewVars := stringMapToInterfaceMap(mergeStringMaps(planConfigVars, planSensitiveConfigVars))

	// Only call updateConfigVars when something actually changed.
	if !mapsEqualStringInterface(allOldVars, allNewVars) {
		if err := updateConfigVars(appID, client, allOldVars, allNewVars); err != nil {
			resp.Diagnostics.AddError("Error updating config vars", err.Error())
			return
		}
	}

	// ---- ACM ----
	if !plan.Acm.Equal(state.Acm) {
		if err := updateAcm(appID, client, plan.Acm.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error updating ACM", err.Error())
			return
		}
	}

	// ---- Organization locked state ----
	if len(plan.Organization) > 0 && len(state.Organization) > 0 {
		if !plan.Organization[0].Locked.Equal(state.Organization[0].Locked) {
			locked := plan.Organization[0].Locked.ValueBool()
			log.Printf("[DEBUG] Organization updated locked: %t", locked)
			teamAppUpdateOpts := heroku.TeamAppUpdateLockedOpts{Locked: locked}
			if _, err := client.TeamAppUpdateLocked(ctx, appID, teamAppUpdateOpts); err != nil {
				resp.Diagnostics.AddError("Error updating team app locked state", err.Error())
				return
			}
		}
	}

	// Read back the full state.
	if err := r.readIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading app after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// -----------------------------------------------------------------------------
// Delete
// -----------------------------------------------------------------------------

func (r *appResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := state.ID.ValueString()
	log.Printf("[INFO] Deleting App: %s", appID)
	if _, err := r.config.Api.AppDelete(ctx, appID); err != nil {
		resp.Diagnostics.AddError("Error deleting Heroku app", fmt.Sprintf("error deleting App: %s", err))
		return
	}
}

// -----------------------------------------------------------------------------
// ImportState
// -----------------------------------------------------------------------------

// ImportState mirrors resourceHerokuAppImport: look up the app by the given ID
// (name or UUID), set the resource ID to the app's UUID, then run a full Read.
func (r *appResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, err := r.config.Api.AppInfo(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching app during import", err.Error())
		return
	}

	// Use the UUID as the resource ID (matches SDKv2 import behaviour).
	state := appResourceModel{
		ID: types.StringValue(app.ID),
	}

	if err := r.readIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading app during import", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// -----------------------------------------------------------------------------
// UpgradeState (v0 → v1)
// -----------------------------------------------------------------------------

// UpgradeState handles the v0→v1 migration for heroku_app. The prior schema
// (version 0) lacked the "generation" attribute. The critical logic mirrors
// resourceHerokuAppStateUpgradeV0: the resource's "id" field is resolved — if
// it is already a UUID it is kept; otherwise the app is looked up by name and
// the UUID is substituted.
func (r *appResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuAppV0(): all attributes present in v1
	// except "generation".
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"space": schema.StringAttribute{
				Optional: true,
			},
			"stack": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"internal_routing": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"buildpacks": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"config_vars": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"sensitive_config_vars": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"all_config_vars": schema.MapAttribute{
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"git_url": schema.StringAttribute{
				Computed: true,
			},
			"web_url": schema.StringAttribute{
				Computed: true,
			},
			"acm": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"heroku_hostname": schema.StringAttribute{
				Computed: true,
			},
			"uuid": schema.StringAttribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"organization": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"locked": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"personal": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior appResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Resolve the "id" field: keep it if it is already a valid UUID;
				// otherwise call the API to fetch the app UUID by name. This is
				// the EXACT logic of resourceHerokuAppStateUpgradeV0.
				appIdentifier := prior.ID.ValueString()
				resolvedID := appIdentifier
				if _, err := uuid.ParseUUID(appIdentifier); err != nil {
					// Not a UUID — look up by name/identifier.
					foundApp, apiErr := r.config.Api.AppInfo(ctx, appIdentifier)
					if apiErr != nil {
						resp.Diagnostics.AddError(
							"Error upgrading heroku_app state",
							fmt.Sprintf("resourceHerokuAppStateUpgradeV0 error retrieving app '%s': %s", appIdentifier, apiErr),
						)
						return
					}
					resolvedID = foundApp.ID
				}

				// Build the upgraded model: copy all prior fields and add the
				// new "generation" attribute as Unknown (will be resolved on the
				// next refresh).
				upgraded := appResourceModel{
					ID:                  types.StringValue(resolvedID),
					Name:                prior.Name,
					Region:              prior.Region,
					Space:               prior.Space,
					Stack:               prior.Stack,
					Generation:          types.StringUnknown(),
					InternalRouting:     prior.InternalRouting,
					Buildpacks:          prior.Buildpacks,
					ConfigVars:          prior.ConfigVars,
					SensitiveConfigVars: prior.SensitiveConfigVars,
					AllConfigVars:       prior.AllConfigVars,
					GitURL:              prior.GitURL,
					WebURL:              prior.WebURL,
					Acm:                 prior.Acm,
					HerokuHostname:      prior.HerokuHostname,
					Organization:        prior.Organization,
					UUID:                prior.UUID,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Private helpers
// -----------------------------------------------------------------------------

// readIntoModel calls resourceHerokuAppRetrieve (shared from
// resource_heroku_app.go) and populates the model, mirroring the logic of
// resourceHerokuAppRead.
func (r *appResource) readIntoModel(ctx context.Context, m *appResourceModel) error {
	appID := m.ID.ValueString()
	client := r.config.Api

	app, err := resourceHerokuAppRetrieve(appID, client)
	if err != nil {
		return err
	}

	m.Name = types.StringValue(app.App.Name)
	m.Stack = types.StringValue(app.App.Stack)
	m.InternalRouting = types.BoolValue(app.App.InternalRouting)
	m.Region = types.StringValue(app.App.Region)
	m.GitURL = types.StringValue(app.App.GitURL)
	m.WebURL = types.StringValue(app.App.WebURL)
	m.Acm = types.BoolValue(app.App.Acm)
	m.UUID = types.StringValue(app.App.ID)
	m.Generation = types.StringValue(app.Generation)

	// heroku_hostname: strip the URL scheme and trailing slash from WebURL,
	// mirroring the SDKv2 setAppDetails() computation.
	hostname := strings.TrimPrefix(
		strings.TrimPrefix(
			strings.TrimSuffix(app.App.WebURL, "/"),
			"https://"),
		"http://")
	m.HerokuHostname = types.StringValue(hostname)

	// Space.
	if app.App.Space != "" {
		m.Space = types.StringValue(app.App.Space)
	} else {
		m.Space = types.StringNull()
	}

	// Organization / team details.
	if app.IsTeamApp {
		// Preserve the "personal" value from prior state since the API does not
		// return it (mirrors the SDKv2 setTeamDetails comment).
		personal := false
		if len(m.Organization) > 0 {
			personal = m.Organization[0].Personal.ValueBool()
		}
		m.Organization = []appOrgModel{
			{
				Name:     types.StringValue(app.App.TeamName),
				Locked:   types.BoolValue(app.App.Locked),
				Personal: types.BoolValue(personal),
			},
		}
	}

	// Buildpacks.
	bpList, bpDiags := types.ListValueFrom(ctx, types.StringType, app.Buildpacks)
	if bpDiags.HasError() {
		log.Printf("[WARN] Error converting buildpacks to list")
	} else {
		m.Buildpacks = bpList
	}

	// Config vars: only surface the keys that are already tracked by the
	// resource (same as resourceHerokuAppRead).
	currentConfigVars := mapFromState(m.ConfigVars)
	currentSensitiveConfigVars := mapFromState(m.SensitiveConfigVars)

	configVars := make(map[string]string)
	for k := range currentConfigVars {
		if v, ok := app.Vars[k]; ok {
			configVars[k] = v
		}
	}

	sensitiveConfigVars := make(map[string]string)
	for k := range currentSensitiveConfigVars {
		if v, ok := app.Vars[k]; ok {
			sensitiveConfigVars[k] = v
		}
	}

	log.Printf("[DEBUG] Setting config vars: %s", configVars)
	configVarsMap, cvDiags := types.MapValueFrom(ctx, types.StringType, configVars)
	if cvDiags.HasError() {
		log.Printf("[WARN] Error setting config vars")
	} else {
		m.ConfigVars = configVarsMap
	}

	sensitiveVarNames := make([]string, 0, len(sensitiveConfigVars))
	for k := range sensitiveConfigVars {
		sensitiveVarNames = append(sensitiveVarNames, k)
	}
	log.Printf("[DEBUG] Setting sensitive config vars: %s", sensitiveVarNames)
	sensVarsMap, svDiags := types.MapValueFrom(ctx, types.StringType, sensitiveConfigVars)
	if svDiags.HasError() {
		log.Printf("[WARN] Error setting sensitive config vars")
	} else {
		m.SensitiveConfigVars = sensVarsMap
	}

	// all_config_vars: empty by default; only populated when
	// SetAppAllConfigVarsInState is true (mirrors resourceHerokuAppRead).
	emptyMap, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	m.AllConfigVars = emptyMap
	if r.config.SetAppAllConfigVarsInState {
		allVarsMap, allDiags := types.MapValueFrom(ctx, types.StringType, app.Vars)
		if allDiags.HasError() {
			log.Printf("[WARN] Error setting all_config_vars")
		} else {
			m.AllConfigVars = allVarsMap
		}
	}

	return nil
}

// isAppNotFound returns true when the error represents a Heroku 404 / not_found.
func isAppNotFound(err error) bool {
	if err == nil {
		return false
	}
	urlErr, ok := err.(*url.Error)
	if !ok {
		return false
	}
	if herr, ok := urlErr.Err.(heroku.Error); ok && herr.ID == "not_found" {
		return true
	}
	return false
}

// appBuildpacksToStringSlice extracts the string values from a types.List of strings.
// Returns nil (not an empty slice) when the list is null or unknown.
func appBuildpacksToStringSlice(l types.List) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	elems := l.Elements()
	result := make([]string, 0, len(elems))
	for _, e := range elems {
		if sv, ok := e.(types.String); ok && !sv.IsNull() && !sv.IsUnknown() {
			result = append(result, sv.ValueString())
		}
	}
	return result
}

// buildpackListsEqual returns true when two buildpack lists contain the same
// elements in the same order.
func buildpackListsEqual(a, b types.List) bool {
	if a.IsNull() != b.IsNull() || a.IsUnknown() != b.IsUnknown() {
		return false
	}
	aElems := a.Elements()
	bElems := b.Elements()
	if len(aElems) != len(bElems) {
		return false
	}
	for i := range aElems {
		if !aElems[i].Equal(bElems[i]) {
			return false
		}
	}
	return true
}

// mapsEqualStringInterface compares two map[string]interface{} for equality
// using string representation of values.
func mapsEqualStringInterface(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", av) != fmt.Sprintf("%v", bv) {
			return false
		}
	}
	return true
}
