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
	_ resource.Resource                 = (*collaboratorResource)(nil)
	_ resource.ResourceWithConfigure    = (*collaboratorResource)(nil)
	_ resource.ResourceWithImportState  = (*collaboratorResource)(nil)
	_ resource.ResourceWithUpgradeState = (*collaboratorResource)(nil)
)

// NewCollaboratorResource returns the framework implementation of the
// heroku_collaborator resource.
func NewCollaboratorResource() resource.Resource {
	return &collaboratorResource{}
}

type collaboratorResource struct {
	config *Config
}

// collaboratorResourceModel is the model for the current schema (version 1).
type collaboratorResourceModel struct {
	ID    types.String `tfsdk:"id"`
	AppID types.String `tfsdk:"app_id"`
	Email types.String `tfsdk:"email"`
}

// collaboratorResourceModelV0 is the model for the prior schema (version 0),
// which used "app" instead of "app_id".
type collaboratorResourceModelV0 struct {
	ID    types.String `tfsdk:"id"`
	App   types.String `tfsdk:"app"`
	Email types.String `tfsdk:"email"`
}

func (r *collaboratorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collaborator"
}

func (r *collaboratorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"email": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *collaboratorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *collaboratorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan collaboratorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.CollaboratorCreateOpts{
		User: plan.Email.ValueString(),
	}

	// Setting the silent parameter to true by default. It is really an optional
	// parameter that doesn't belong in the resource's state, especially since
	// it's not part of the collaborator GET endpoint.
	vs := true
	opts.Silent = &vs

	log.Printf("[DEBUG] Creating Heroku Collaborator: [%s]", opts.User)
	created, err := r.config.Api.CollaboratorCreate(ctx, plan.AppID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating collaborator", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	log.Printf("[INFO] New Collaborator ID: %s", plan.ID.ValueString())

	if err := r.readCollaborator(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading collaborator after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *collaboratorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state collaboratorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readCollaborator(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading collaborator", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op because all fields are ForceNew (RequiresReplace), so
// any change triggers a destroy+create cycle and Update is never called.
// The method must exist to satisfy the resource.Resource interface.
func (r *collaboratorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state collaboratorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *collaboratorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state collaboratorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting Heroku Collaborator: [%s]", state.ID.ValueString())
	if _, err := r.config.Api.CollaboratorDelete(ctx, state.AppID.ValueString(), state.Email.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting collaborator",
			fmt.Sprintf("error deleting Collaborator: %s", err),
		)
		return
	}

	// After DELETE succeeds, poll until the collaborator is gone. This mirrors
	// the SDKv2 retry pattern: the Heroku API can return stale data immediately
	// after deletion, causing a subsequent create to fail with a conflict error.
	log.Printf("[INFO] Begin checking if [%s] has been deleted", state.Email.ValueString())
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_, err := r.config.Api.CollaboratorInfo(ctx, state.AppID.ValueString(), state.ID.ValueString())
		log.Printf("[INFO] Is error nil when GET#show collaborator? %t", err == nil)
		if err != nil {
			// Non-nil error means the collaborator is gone — success.
			return
		}
		log.Printf("[WARNING] Collaborator [%s] exists after deletion. Checking again", state.Email.ValueString())
		time.Sleep(1 * time.Second)
	}

	resp.Diagnostics.AddError(
		"Error confirming collaborator deletion",
		fmt.Sprintf("[ERROR] Collaborator [%s] still exists on [%s] after checking several times", state.Email.ValueString(), state.AppID.ValueString()),
	)
}

func (r *collaboratorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, email, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing collaborator", err.Error())
		return
	}

	collaboratorInfo, err := r.config.Api.CollaboratorInfo(ctx, app, email)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching collaborator during import", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), collaboratorInfo.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), collaboratorInfo.App.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), collaboratorInfo.User.Email)...)
}

// UpgradeState handles the v0→v1 migration that renames "app" to "app_id"
// (resolving app name to UUID if needed), mirroring the SDKv2 upgradeAppToAppID
// state upgrader registered against resourceHerokuCollaboratorV0().
func (r *collaboratorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuCollaboratorV0(): "app" + "email" (no app_id).
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"email": schema.StringAttribute{
				Required: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior collaboratorResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading collaborator state", err.Error())
					return
				}

				upgraded := collaboratorResourceModel{
					ID:    prior.ID,
					AppID: types.StringValue(appID),
					Email: prior.Email,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// readCollaborator fetches current state from the API and populates the model.
// It mirrors the SDKv2 resourceHerokuCollaboratorRead logic.
func (r *collaboratorResource) readCollaborator(ctx context.Context, m *collaboratorResourceModel) error {
	collaboratorInfo, err := r.config.Api.CollaboratorInfo(ctx, m.AppID.ValueString(), m.ID.ValueString())
	if err != nil {
		return fmt.Errorf("[ERROR] Error retrieving collaborator: %s", err)
	}

	m.AppID = types.StringValue(collaboratorInfo.App.ID)
	m.Email = types.StringValue(collaboratorInfo.User.Email)
	return nil
}
