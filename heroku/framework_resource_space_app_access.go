package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*spaceAppAccessResource)(nil)
	_ resource.ResourceWithConfigure   = (*spaceAppAccessResource)(nil)
	_ resource.ResourceWithImportState = (*spaceAppAccessResource)(nil)
)

// NewSpaceAppAccessResource returns the framework implementation of the
// heroku_space_app_access resource.
func NewSpaceAppAccessResource() resource.Resource {
	return &spaceAppAccessResource{}
}

type spaceAppAccessResource struct {
	config *Config
}

type spaceAppAccessResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Space       types.String `tfsdk:"space"`
	Email       types.String `tfsdk:"email"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func (r *spaceAppAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_app_access"
}

func (r *spaceAppAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"space": schema.StringAttribute{
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

func (r *spaceAppAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *spaceAppAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan spaceAppAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.setSpaceAppAccess(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating space app access", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *spaceAppAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state spaceAppAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readSpaceAppAccess(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading space app access", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *spaceAppAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan spaceAppAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	var state spaceAppAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	if err := r.setSpaceAppAccess(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating space app access", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete has no real Heroku API removal — members cannot be deleted from a
// space via this resource. Permissions are cleared out and the resource is
// removed from state, mirroring the SDKv2 behaviour.
func (r *spaceAppAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state spaceAppAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clear all permissions by passing an empty slice.
	opts := heroku.SpaceAppAccessUpdateOpts{
		Permissions: []struct {
			Name *string `json:"name,omitempty" url:"name,omitempty,key"`
		}{},
	}
	if _, err := r.config.Api.SpaceAppAccessUpdate(ctx, state.Space.ValueString(), state.Email.ValueString(), opts); err != nil {
		resp.Diagnostics.AddError("Error clearing space app access permissions", err.Error())
		return
	}
}

func (r *spaceAppAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	space, email, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing space app access", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("space"), space)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email"), email)...)
}

// setSpaceAppAccess calls SpaceAppAccessUpdate then reads current state back
// into the model, mirroring the SDKv2 pattern of calling
// resourceHerokuSpaceAppAccessRead after the write.
func (r *spaceAppAccessResource) setSpaceAppAccess(ctx context.Context, m *spaceAppAccessResourceModel) error {
	var permNames []string
	if diags := m.Permissions.ElementsAs(ctx, &permNames, false); diags.HasError() {
		return fmt.Errorf("error reading permissions from plan")
	}

	opts := heroku.SpaceAppAccessUpdateOpts{
		Permissions: make([]struct {
			Name *string `json:"name,omitempty" url:"name,omitempty,key"`
		}, 0, len(permNames)),
	}
	for i := range permNames {
		name := permNames[i]
		opts.Permissions = append(opts.Permissions, struct {
			Name *string `json:"name,omitempty" url:"name,omitempty,key"`
		}{Name: &name})
	}

	if _, err := r.config.Api.SpaceAppAccessUpdate(ctx, m.Space.ValueString(), m.Email.ValueString(), opts); err != nil {
		return err
	}

	return r.readSpaceAppAccess(ctx, m)
}

// readSpaceAppAccess fetches current access info and populates the model,
// mirroring resourceHerokuSpaceAppAccessRead.
func (r *spaceAppAccessResource) readSpaceAppAccess(ctx context.Context, m *spaceAppAccessResourceModel) error {
	spaceAppAccess, err := r.config.Api.SpaceAppAccessInfo(ctx, m.Space.ValueString(), m.Email.ValueString())
	if err != nil {
		return err
	}

	// The SDKv2 resource sets the ID to User.ID on reads.
	m.ID = types.StringValue(spaceAppAccess.User.ID)
	m.Email = types.StringValue(spaceAppAccess.User.Email)

	permElems := make([]attr.Value, 0, len(spaceAppAccess.Permissions))
	for _, perm := range spaceAppAccess.Permissions {
		permElems = append(permElems, types.StringValue(perm.Name))
	}
	permSet, diags := types.SetValue(types.StringType, permElems)
	if diags.HasError() {
		return fmt.Errorf("error building permissions set")
	}
	m.Permissions = permSet

	return nil
}
