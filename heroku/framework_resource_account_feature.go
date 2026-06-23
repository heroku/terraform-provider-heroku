package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*accountFeatureResource)(nil)
	_ resource.ResourceWithConfigure   = (*accountFeatureResource)(nil)
	_ resource.ResourceWithImportState = (*accountFeatureResource)(nil)
)

// NewAccountFeatureResource returns the framework implementation of the
// heroku_account_feature resource.
func NewAccountFeatureResource() resource.Resource {
	return &accountFeatureResource{}
}

type accountFeatureResource struct {
	config *Config
}

type accountFeatureResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
}

func (r *accountFeatureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_feature"
}

func (r *accountFeatureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"state": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *accountFeatureResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accountFeatureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accountFeatureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	feature, err := r.updateAccountFeature(ctx, plan.Name.ValueString(), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error enabling/disabling account feature", err.Error())
		return
	}

	account, err := r.config.Api.AccountInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Heroku account", err.Error())
		return
	}

	plan.ID = types.StringValue(buildCompositeID(account.Email, feature.Name))
	r.setModelFromFeature(&plan, feature)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *accountFeatureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accountFeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	feature, err := r.config.Api.AccountFeatureInfo(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading account feature", err.Error())
		return
	}

	r.setModelFromFeature(&state, feature)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accountFeatureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accountFeatureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	feature, err := r.updateAccountFeature(ctx, plan.Name.ValueString(), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error enabling/disabling account feature", err.Error())
		return
	}

	r.setModelFromFeature(&plan, feature)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete has no Heroku API endpoint, so it disables the feature and lets the
// framework remove the resource from state.
func (r *accountFeatureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accountFeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.updateAccountFeature(ctx, state.Name.ValueString(), false); err != nil {
		resp.Diagnostics.AddError("Error disabling account feature", err.Error())
		return
	}
}

func (r *accountFeatureResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	_, featureName, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing account feature", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), featureName)...)
}

func (r *accountFeatureResource) updateAccountFeature(ctx context.Context, name string, enabled bool) (*heroku.AccountFeature, error) {
	opts := heroku.AccountFeatureUpdateOpts{Enabled: enabled}
	feature, err := r.config.Api.AccountFeatureUpdate(ctx, name, opts)
	if err != nil {
		return nil, fmt.Errorf("opts %+v: %w", opts, err)
	}
	return feature, nil
}

func (r *accountFeatureResource) setModelFromFeature(m *accountFeatureResourceModel, feature *heroku.AccountFeature) {
	m.Name = types.StringValue(feature.Name)
	m.Description = types.StringValue(feature.Description)
	m.State = types.StringValue(feature.State)
	m.Enabled = types.BoolValue(feature.Enabled)
}
