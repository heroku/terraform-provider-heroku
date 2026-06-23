package heroku

import (
	"context"
	"fmt"

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
	_ resource.Resource                = (*teamMemberResource)(nil)
	_ resource.ResourceWithConfigure   = (*teamMemberResource)(nil)
	_ resource.ResourceWithImportState = (*teamMemberResource)(nil)
)

// NewTeamMemberResource returns the framework implementation of the
// heroku_team_member resource.
func NewTeamMemberResource() resource.Resource {
	return &teamMemberResource{}
}

type teamMemberResource struct {
	config *Config
}

type teamMemberResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Team      types.String `tfsdk:"team"`
	Email     types.String `tfsdk:"email"`
	Role      types.String `tfsdk:"role"`
	Federated types.Bool   `tfsdk:"federated"`
}

func (r *teamMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_member"
}

func (r *teamMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team": schema.StringAttribute{
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
			"role": schema.StringAttribute{
				Required: true,
			},
			"federated": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *teamMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.setTeamMember(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating team member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readTeamMember(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading team member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	var state teamMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	if err := r.setTeamMember(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating team member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	team, email, err := parseCompositeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing team member ID", err.Error())
		return
	}

	if _, err := r.config.Api.TeamMemberDelete(ctx, team, email); err != nil {
		resp.Diagnostics.AddError("Error deleting team member", err.Error())
		return
	}
}

func (r *teamMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	team, email, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing team member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), team)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), email)...)
}

// setTeamMember calls TeamMemberCreateOrUpdate then reads current state back
// into the model, mirroring the SDKv2 pattern of calling resourceHerokuTeamMemberRead
// after the write.
func (r *teamMemberResource) setTeamMember(ctx context.Context, m *teamMemberResourceModel) error {
	federated := m.Federated.ValueBool()
	opts := heroku.TeamMemberCreateOrUpdateOpts{
		Email:     m.Email.ValueString(),
		Role:      m.Role.ValueString(),
		Federated: &federated,
	}

	if _, err := r.config.Api.TeamMemberCreateOrUpdate(ctx, m.Team.ValueString(), opts); err != nil {
		return err
	}

	m.ID = types.StringValue(buildCompositeID(m.Team.ValueString(), m.Email.ValueString()))
	return r.readTeamMember(ctx, m)
}

// readTeamMember lists team members and populates the model from the matching
// entry, mirroring the SDKv2 resourceHerokuTeamMemberRead logic exactly.
func (r *teamMemberResource) readTeamMember(ctx context.Context, m *teamMemberResourceModel) error {
	team, email, err := parseCompositeID(m.ID.ValueString())
	if err != nil {
		return err
	}

	members, err := r.config.Api.TeamMemberList(ctx, team, &heroku.ListRange{Field: "email", Max: 1000})
	if err != nil {
		return err
	}

	var found heroku.TeamMember
	for _, member := range members {
		if member.Email == email {
			found = member
			break
		}
	}

	if found.ID == "" {
		return fmt.Errorf("could not find member record for %s on team %s", email, team)
	}

	m.Team = types.StringValue(team)
	m.Email = types.StringValue(found.Email)
	m.Federated = types.BoolValue(found.Federated)
	if found.Role != nil {
		m.Role = types.StringValue(*found.Role)
	}

	return nil
}
