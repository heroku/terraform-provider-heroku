package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*appReleaseResource)(nil)
	_ resource.ResourceWithConfigure    = (*appReleaseResource)(nil)
	_ resource.ResourceWithImportState  = (*appReleaseResource)(nil)
	_ resource.ResourceWithUpgradeState = (*appReleaseResource)(nil)
)

// NewAppReleaseResource returns the framework implementation of the
// heroku_app_release resource.
func NewAppReleaseResource() resource.Resource {
	return &appReleaseResource{}
}

type appReleaseResource struct {
	config *Config
}

// appReleaseResourceModel is the model for the current schema (version 1).
type appReleaseResourceModel struct {
	ID          types.String `tfsdk:"id"`
	AppID       types.String `tfsdk:"app_id"`
	SlugID      types.String `tfsdk:"slug_id"`
	OciImage    types.String `tfsdk:"oci_image"`
	Description types.String `tfsdk:"description"`
}

// appReleaseResourceModelV0 is the model for the prior schema (version 0),
// which used "app" instead of "app_id" and only had "slug_id".
type appReleaseResourceModelV0 struct {
	ID          types.String `tfsdk:"id"`
	App         types.String `tfsdk:"app"`
	SlugID      types.String `tfsdk:"slug_id"`
	Description types.String `tfsdk:"description"`
}

func (r *appReleaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_release"
}

func (r *appReleaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"slug_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"oci_image": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *appReleaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *appReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appReleaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.ReleaseCreateOpts{}

	appID := plan.AppID.ValueString()

	if !plan.SlugID.IsNull() && !plan.SlugID.IsUnknown() {
		vs := plan.SlugID.ValueString()
		log.Printf("[DEBUG] Slug Id: %s", vs)
		opts.Slug = vs
	}

	if !plan.OciImage.IsNull() && !plan.OciImage.IsUnknown() {
		vs := plan.OciImage.ValueString()
		log.Printf("[DEBUG] OCI Image: %s", vs)
		opts.OciImage = &vs
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		vs := plan.Description.ValueString()
		log.Printf("[DEBUG] description: %s", vs)
		opts.Description = &vs
	}

	log.Printf("[DEBUG] Creating a new release on app: [%s]", appID)
	newRelease, err := r.config.Api.ReleaseCreate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating app release", err.Error())
		return
	}

	log.Printf("[INFO] New release ID: %s", newRelease.ID)
	log.Printf("[INFO] Begin Checking if new Release %s is successful", newRelease.ID)

	if err := waitForReleaseSucceeded(ctx, r.config.Api, appID, newRelease.ID, 20*time.Minute); err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for app release",
			fmt.Sprintf("[ERROR] Error waiting for new release (%s) to succeed: %s", newRelease.ID, err),
		)
		return
	}

	plan.ID = types.StringValue(newRelease.ID)

	if err := r.readAppRelease(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading app release after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appReleaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readAppRelease(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading app release", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op because all writable fields (app_id, slug_id, oci_image)
// are ForceNew (RequiresReplace). The description field is Computed and cannot
// be changed on an existing release. This mirrors the SDKv2
// resourceHerokuAppReleaseUpdate which is also effectively a no-op.
func (r *appReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state appReleaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete is a no-op because there is no DELETE endpoint for the release
// resource in the Heroku Platform APIs. The resource is simply removed from
// state, mirroring the SDKv2 resourceHerokuAppReleaseDelete.
func (r *appReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	log.Printf("[INFO] There is no DELETE for release resource so this is a no-op. Resource will be removed from state.")
}

func (r *appReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the app name or ID. We fetch the latest current release
	// for that app, mirroring the SDKv2 resourceHerokuAppReleaseImport logic.
	appName := req.ID

	log.Printf("[INFO] Importing Release for App [%s]", appName)

	appReleases, err := r.config.Api.ReleaseList(ctx, appName, &heroku.ListRange{Descending: true, Field: "version", Max: 1})
	if err != nil {
		resp.Diagnostics.AddError("Error listing releases during import", err.Error())
		return
	}

	if len(appReleases) == 0 {
		resp.Diagnostics.AddError("Error importing app release", fmt.Sprintf("no releases found for app [%s]", appName))
		return
	}

	appRelease := appReleases[0]

	if !appRelease.Current {
		resp.Diagnostics.AddError(
			"Error importing app release",
			fmt.Sprintf("[ERROR] The latest release for app [%s] is not current for some reason", appName),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), appRelease.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appRelease.App.ID)...)

	if appRelease.Slug != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug_id"), appRelease.Slug.ID)...)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), appRelease.Description)...)
}

// UpgradeState handles the v0→v1 migration that renames "app" to "app_id"
// (resolving app name to UUID if needed), mirroring the SDKv2 upgradeAppToAppID
// state upgrader registered against resourceHerokuAppReleaseV0().
func (r *appReleaseResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuAppReleaseV0(): "app" + "slug_id" (no app_id, no oci_image).
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"slug_id": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior appReleaseResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading app release state", err.Error())
					return
				}

				upgraded := appReleaseResourceModel{
					ID:          prior.ID,
					AppID:       types.StringValue(appID),
					SlugID:      prior.SlugID,
					OciImage:    types.StringNull(),
					Description: prior.Description,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// readAppRelease fetches current state from the API and populates the model.
// It mirrors the SDKv2 resourceHerokuAppReleaseRead logic.
func (r *appReleaseResource) readAppRelease(ctx context.Context, m *appReleaseResourceModel) error {
	appRelease, err := r.config.Api.ReleaseInfo(ctx, m.AppID.ValueString(), m.ID.ValueString())
	if err != nil {
		return fmt.Errorf("[ERROR] error retrieving app release: %s", err)
	}

	m.AppID = types.StringValue(appRelease.App.ID)

	// Handle Cedar releases (with slugs).
	if appRelease.Slug != nil {
		m.SlugID = types.StringValue(appRelease.Slug.ID)
	}

	// Handle Fir releases (with OCI images).
	for _, artifact := range appRelease.Artifacts {
		if artifact.Type == "oci-image" {
			m.OciImage = types.StringValue(artifact.ID)
			break
		}
	}

	m.Description = types.StringValue(appRelease.Description)

	return nil
}
