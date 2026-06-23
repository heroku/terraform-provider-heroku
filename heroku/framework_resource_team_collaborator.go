package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	_ resource.Resource                 = (*teamCollaboratorResource)(nil)
	_ resource.ResourceWithConfigure    = (*teamCollaboratorResource)(nil)
	_ resource.ResourceWithImportState  = (*teamCollaboratorResource)(nil)
	_ resource.ResourceWithUpgradeState = (*teamCollaboratorResource)(nil)
)

// NewTeamCollaboratorResource returns the framework implementation of the
// heroku_team_collaborator resource.
func NewTeamCollaboratorResource() resource.Resource {
	return &teamCollaboratorResource{}
}

type teamCollaboratorResource struct {
	config *Config
}

type teamCollaboratorResourceModel struct {
	ID          types.String `tfsdk:"id"`
	AppID       types.String `tfsdk:"app_id"`
	Email       types.String `tfsdk:"email"`
	Permissions types.Set    `tfsdk:"permissions"`
}

// teamCollaboratorResourceV0Model mirrors resourceHerokuTeamCollaboratorV0() for
// the state upgrade from schema version 0.
type teamCollaboratorResourceV0Model struct {
	ID          types.String `tfsdk:"id"`
	App         types.String `tfsdk:"app"`
	Email       types.String `tfsdk:"email"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func (r *teamCollaboratorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_collaborator"
}

func (r *teamCollaboratorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"permissions": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *teamCollaboratorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamCollaboratorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamCollaboratorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()

	var permStrs []string
	resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permStrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms := make([]*string, len(permStrs))
	for i := range permStrs {
		p := permStrs[i]
		perms[i] = &p
	}

	vs := true
	opts := heroku.TeamAppCollaboratorCreateOpts{
		User:        plan.Email.ValueString(),
		Silent:      &vs,
		Permissions: perms,
	}

	log.Printf("[DEBUG] Creating Heroku Team Collaborator: [%s]", opts.User)

	var resourceID string
	collaborator, createErr := r.config.Api.TeamAppCollaboratorCreate(ctx, appID, opts)
	if createErr != nil {
		if strings.Contains(strings.ToLower(createErr.Error()), "is already a collaborator on app") {
			collaborators, listErr := r.config.Api.TeamAppCollaboratorList(ctx, appID,
				&heroku.ListRange{Field: "email", Max: 1000})
			if listErr != nil {
				resp.Diagnostics.AddError("Error listing team collaborators", listErr.Error())
				return
			}
			for _, c := range collaborators {
				if c.User.Email == opts.User {
					resourceID = c.ID
					break
				}
			}
			if resourceID == "" {
				resp.Diagnostics.AddError("Error creating team collaborator",
					fmt.Sprintf("collaborator already exists but could not be found in list for app %s", appID))
				return
			}
		} else {
			resp.Diagnostics.AddError("Error creating team collaborator", createErr.Error())
			return
		}
	} else {
		resourceID = collaborator.ID
	}

	plan.ID = types.StringValue(resourceID)
	log.Printf("[INFO] New Team Collaborator ID: %s", resourceID)

	if err := r.readTeamCollaborator(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading team collaborator after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamCollaboratorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamCollaboratorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readTeamCollaborator(ctx, &state)
	if err != nil {
		if strings.Contains(err.Error(), "Couldn't find that user") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading team collaborator", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamCollaboratorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamCollaboratorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state teamCollaboratorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	var permStrs []string
	resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permStrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.TeamAppCollaboratorUpdateOpts{
		Permissions: permStrs,
	}

	appID := plan.AppID.ValueString()
	email := plan.Email.ValueString()

	log.Printf("[DEBUG] Updating Heroku Team Collaborator: [%s]", email)
	updated, err := r.config.Api.TeamAppCollaboratorUpdate(ctx, appID, email, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating team collaborator", err.Error())
		return
	}

	plan.ID = types.StringValue(updated.ID)

	if err := r.readTeamCollaborator(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading team collaborator after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamCollaboratorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamCollaboratorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := state.AppID.ValueString()
	email := state.Email.ValueString()
	collaboratorID := state.ID.ValueString()

	log.Printf("[INFO] Deleting Heroku Team Collaborator: [%s]", collaboratorID)
	_, err := r.config.Api.TeamAppCollaboratorDelete(ctx, appID, email)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting team collaborator",
			fmt.Sprintf("error deleting Team Collaborator: %s", err))
		return
	}

	// Poll until the collaborator is fully removed to avoid immediate-recreate race.
	log.Printf("[INFO] Begin checking if [%s] has been deleted", email)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_, checkErr := r.config.Api.TeamAppCollaboratorInfo(ctx, appID, collaboratorID)
		log.Printf("[INFO] Is error nil when GET#show team collaborator? %t", checkErr == nil)
		if checkErr != nil {
			// Collaborator no longer found — deletion confirmed.
			break
		}
		log.Printf("[WARNING] Team collaborator [%s] exists after deletion. Checking again", email)
		time.Sleep(1 * time.Second)
	}
}

func (r *teamCollaboratorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, email, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing team collaborator", err.Error())
		return
	}

	collaborator, err := r.config.Api.CollaboratorInfo(ctx, app, email)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team collaborator during import", err.Error())
		return
	}

	var permNames []string
	for _, p := range collaborator.Permissions {
		permNames = append(permNames, p.Name)
	}
	permSet, diags := types.SetValueFrom(ctx, types.StringType, permNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), collaborator.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), collaborator.App.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), collaborator.User.Email)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permissions"), permSet)...)
}

// UpgradeState implements resource.ResourceWithUpgradeState.
// It mirrors the SDKv2 upgradeAppToAppID upgrader: v0 used "app" (name or UUID);
// v1 replaced it with "app_id" (UUID).
func (r *teamCollaboratorResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
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
					"permissions": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior teamCollaboratorResourceV0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading team collaborator state", err.Error())
					return
				}

				upgraded := teamCollaboratorResourceModel{
					ID:          prior.ID,
					AppID:       types.StringValue(appID),
					Email:       prior.Email,
					Permissions: prior.Permissions,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// readTeamCollaborator retrieves the collaborator from the Heroku API and
// populates the model. It mirrors the SDKv2 resourceHerokuTeamCollaboratorRead
// logic: TeamAppCollaboratorInfo for identity/app fields, CollaboratorInfo for
// permissions (the go client does not return permissions via the team endpoint).
func (r *teamCollaboratorResource) readTeamCollaborator(ctx context.Context, m *teamCollaboratorResourceModel) error {
	appID := m.AppID.ValueString()
	collaboratorID := m.ID.ValueString()

	tcInfo, err := r.config.Api.TeamAppCollaboratorInfo(ctx, appID, collaboratorID)
	if err != nil {
		return err
	}

	m.Email = types.StringValue(tcInfo.User.Email)
	m.AppID = types.StringValue(tcInfo.App.ID)

	// CollaboratorInfo returns permission details not available via TeamAppCollaboratorInfo.
	cInfo, err := r.config.Api.CollaboratorInfo(ctx, appID, collaboratorID)
	if err != nil {
		return err
	}

	permNames := make([]string, 0, len(cInfo.Permissions))
	for _, p := range cInfo.Permissions {
		permNames = append(permNames, p.Name)
	}

	permSet, diags := types.SetValueFrom(ctx, types.StringType, permNames)
	if diags.HasError() {
		return fmt.Errorf("error building permissions set: %v", diags)
	}
	m.Permissions = permSet

	return nil
}
