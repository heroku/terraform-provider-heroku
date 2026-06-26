package heroku

import (
	"context"
	"fmt"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*appFeatureResource)(nil)
	_ resource.ResourceWithConfigure    = (*appFeatureResource)(nil)
	_ resource.ResourceWithImportState  = (*appFeatureResource)(nil)
	_ resource.ResourceWithUpgradeState = (*appFeatureResource)(nil)
)

// NewAppFeatureResource returns the framework implementation of the
// heroku_app_feature resource.
func NewAppFeatureResource() resource.Resource {
	return &appFeatureResource{}
}

type appFeatureResource struct {
	config *Config
}

// appFeatureResourceModel is the model for the current schema (version 1).
type appFeatureResourceModel struct {
	ID      types.String `tfsdk:"id"`
	AppID   types.String `tfsdk:"app_id"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

// appFeatureResourceModelV0 is the model for the prior schema (version 0),
// which used "app" instead of "app_id".
type appFeatureResourceModelV0 struct {
	ID      types.String `tfsdk:"id"`
	App     types.String `tfsdk:"app"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (r *appFeatureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_feature"
}

func (r *appFeatureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				// Restore SDKv2 validation.IsUUID on app_id (lost in migration).
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
	}
}

func (r *appFeatureResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *appFeatureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appFeatureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsertAppFeature(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating app feature", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appFeatureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appFeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readAppFeature(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading app feature", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update mirrors the SDKv2 resourceHerokuAppFeatureUpdate: when "enabled" changes
// it re-runs the same AppFeatureUpdate+read cycle as Create; otherwise it's a
// read-only refresh. The ID is preserved from state across the update.
func (r *appFeatureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appFeatureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing composite ID from state.
	var state appFeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	if err := r.upsertAppFeature(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating app feature", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete mirrors the SDKv2 resourceHerokuAppFeatureDelete: it disables the
// feature (sets Enabled=false) rather than destroying it, because Heroku app
// features cannot be truly deleted via the API.
func (r *appFeatureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appFeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, id, err := parseCompositeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing app feature ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting app feature %s (%s) for app %s", state.Name.ValueString(), id, app)
	opts := heroku.AppFeatureUpdateOpts{Enabled: false}
	if _, err := r.config.Api.AppFeatureUpdate(ctx, app, id, opts); err != nil {
		resp.Diagnostics.AddError("Error deleting app feature", err.Error())
		return
	}
}

func (r *appFeatureResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, featureID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing app feature", err.Error())
		return
	}

	feature, err := r.config.Api.AppFeatureInfo(ctx, app, featureID)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching app feature during import", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), buildCompositeID(app, feature.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), app)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), feature.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enabled"), feature.Enabled)...)
}

// UpgradeState handles the v0→v1 migration that renames "app" to "app_id"
// (resolving app name to UUID if needed) and rebuilds the composite ID using
// the app UUID, mirroring the SDKv2 upgradeHerokuAppFeatureV1 logic.
func (r *appFeatureResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuAppFeatureV0(): "app" + "name" + "enabled" (no app_id).
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior appFeatureResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading app feature state", err.Error())
					return
				}

				// Rebuild the composite ID: replace the app name/old-id component
				// with the resolved app UUID, keeping the feature UUID component.
				_, featureID, err := parseCompositeID(prior.ID.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Error parsing prior app feature ID during upgrade", err.Error())
					return
				}

				upgraded := appFeatureResourceModel{
					ID:      types.StringValue(buildCompositeID(appID, featureID)),
					AppID:   types.StringValue(appID),
					Name:    prior.Name,
					Enabled: prior.Enabled,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// upsertAppFeature calls AppFeatureUpdate then reads current state back into the
// model, mirroring the SDKv2 pattern of calling resourceHerokuAppFeatureCreate
// (which calls AppFeatureUpdate) followed by resourceHerokuAppFeatureRead.
func (r *appFeatureResource) upsertAppFeature(ctx context.Context, m *appFeatureResourceModel) error {
	enabled := m.Enabled.ValueBool()
	opts := heroku.AppFeatureUpdateOpts{Enabled: enabled}

	log.Printf("[DEBUG] Feature set configuration: %#v, %#v", m.Name.ValueString(), opts)

	feature, err := r.config.Api.AppFeatureUpdate(ctx, m.AppID.ValueString(), m.Name.ValueString(), opts)
	if err != nil {
		return err
	}

	m.ID = types.StringValue(buildCompositeID(m.AppID.ValueString(), feature.ID))
	return r.readAppFeature(ctx, m)
}

// readAppFeature fetches current state from the API and populates the model,
// mirroring the SDKv2 resourceHerokuAppFeatureRead logic.
func (r *appFeatureResource) readAppFeature(ctx context.Context, m *appFeatureResourceModel) error {
	app, id, err := parseCompositeID(m.ID.ValueString())
	if err != nil {
		return err
	}

	feature, err := r.config.Api.AppFeatureInfo(ctx, app, id)
	if err != nil {
		return err
	}

	m.AppID = types.StringValue(app)
	m.Name = types.StringValue(feature.Name)
	m.Enabled = types.BoolValue(feature.Enabled)
	return nil
}
