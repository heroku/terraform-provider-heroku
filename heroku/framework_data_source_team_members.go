package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

// TeamMemberRoles is the set of valid team member roles accepted by the
// heroku_team_members "roles" filter. Restored from the original SDKv2 data
// source (validation.StringInSlice(TeamMemberRoles, false)).
var TeamMemberRoles = []string{"admin", "member", "viewer", "collaborator", "owner"}

// Ensure interface compliance at compile time.
var (
	_ datasource.DataSource              = (*teamMembersDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*teamMembersDataSource)(nil)
)

// NewTeamMembersDataSource returns the framework implementation of the
// heroku_team_members data source.
func NewTeamMembersDataSource() datasource.DataSource {
	return &teamMembersDataSource{}
}

type teamMembersDataSource struct {
	config *Config
}

// teamMemberModel represents a single member entry in the nested members list.
type teamMemberModel struct {
	TeamMemberID            types.String `tfsdk:"team_member_id"`
	UserID                  types.String `tfsdk:"user_id"`
	Email                   types.String `tfsdk:"email"`
	Role                    types.String `tfsdk:"role"`
	Federated               types.Bool   `tfsdk:"federated"`
	TwoFactorAuthentication types.Bool   `tfsdk:"two_factor_authentication"`
}

// teamMembersDataSourceModel is the top-level state model for heroku_team_members.
type teamMembersDataSourceModel struct {
	ID      types.String      `tfsdk:"id"`
	Team    types.String      `tfsdk:"team"`
	Roles   types.List        `tfsdk:"roles"`
	Members []teamMemberModel `tfsdk:"members"`
}

func (d *teamMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (d *teamMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"team": schema.StringAttribute{
				Required: true,
			},
			"roles": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					// Restore the SDKv2 MinItems: 1 and per-element
					// StringInSlice(TeamMemberRoles, false) validation. Without
					// these, an empty filter or a typo'd role is accepted and
					// silently matches no members.
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.OneOf(TeamMemberRoles...)),
				},
			},
			"members": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"team_member_id": schema.StringAttribute{
							Computed: true,
						},
						"user_id": schema.StringAttribute{
							Computed: true,
						},
						"email": schema.StringAttribute{
							Computed: true,
						},
						"role": schema.StringAttribute{
							Computed: true,
						},
						"federated": schema.BoolAttribute{
							Computed: true,
						},
						"two_factor_authentication": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *teamMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.config = config
}

func (d *teamMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model teamMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamName := model.Team.ValueString()

	// Extract the roles filter from the framework List type.
	var roles []string
	resp.Diagnostics.Append(model.Roles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamMembers, err := d.config.Api.TeamMemberList(ctx, teamName, &heroku.ListRange{
		Field:      "id",
		Max:        1000,
		Descending: false,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to retrieve members for team %s", teamName),
			err.Error(),
		)
		return
	}

	if len(teamMembers) == 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("No members found for team %s", teamName),
			"The API returned an empty member list for this team.",
		)
		return
	}

	members := make([]teamMemberModel, 0)
	for _, m := range teamMembers {
		if m.Role == nil {
			continue
		}
		if !SliceContainsString(roles, *m.Role) {
			continue
		}
		members = append(members, teamMemberModel{
			TeamMemberID:            types.StringValue(m.ID),
			UserID:                  types.StringValue(m.User.ID),
			Email:                   types.StringValue(m.User.Email),
			Role:                    types.StringValue(*m.Role),
			Federated:               types.BoolValue(m.Federated),
			TwoFactorAuthentication: types.BoolValue(m.TwoFactorAuthentication),
		})
	}

	model.ID = types.StringValue(teamName)
	model.Members = members

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
