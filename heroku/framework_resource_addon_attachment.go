package heroku

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*addonAttachmentResource)(nil)
	_ resource.ResourceWithConfigure    = (*addonAttachmentResource)(nil)
	_ resource.ResourceWithImportState  = (*addonAttachmentResource)(nil)
	_ resource.ResourceWithUpgradeState = (*addonAttachmentResource)(nil)
)

// NewAddonAttachmentResource returns the framework implementation of the
// heroku_addon_attachment resource.
func NewAddonAttachmentResource() resource.Resource {
	return &addonAttachmentResource{}
}

type addonAttachmentResource struct {
	config *Config
}

type addonAttachmentResourceModel struct {
	ID        types.String `tfsdk:"id"`
	AppID     types.String `tfsdk:"app_id"`
	AddonID   types.String `tfsdk:"addon_id"`
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

func (r *addonAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_addon_attachment"
}

func (r *addonAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version mirrors the SDKv2 SchemaVersion: 1. Without it the framework
		// defaults to version 0, which Terraform treats as a downgrade from any
		// state written by the shipped SDKv2 provider and rejects before refresh.
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
			"addon_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *addonAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *addonAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan addonAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.AddOnAttachmentCreateOpts{
		Addon: plan.AddonID.ValueString(),
		App:   plan.AppID.ValueString(),
	}

	if v := plan.Name.ValueString(); v != "" {
		opts.Name = &v
	}

	if ns := plan.Namespace.ValueString(); ns != "" {
		opts.Namespace = &ns
	}

	a, err := r.config.Api.AddOnAttachmentCreate(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating addon attachment", err.Error())
		return
	}

	plan.ID = types.StringValue(a.ID)

	if err := r.readAddonAttachment(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading addon attachment after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addonAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state addonAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readAddonAttachment(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading addon attachment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op because all attributes are ForceNew; the framework will
// destroy and re-create the resource instead.
func (r *addonAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan addonAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *addonAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state addonAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.config.Api.AddOnAttachmentDelete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting addon attachment", err.Error())
		return
	}
}

func (r *addonAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	matched, _ := regexp.MatchString(`^[0-9a-f]+-[0-9a-f]+-[0-9a-f]+-[0-9a-f]+-[0-9a-f]+$`, req.ID)
	if !matched {
		resp.Diagnostics.AddError(
			"Error importing addon attachment",
			"You can only import addon attachments by their unique ID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// UpgradeState ports the SDKv2 SchemaVersion 1 migration
// (resourceHerokuAddonAttachmentMigrateState). The v0 -> v1 upgrade resolves
// addon_id from the attachment's stored id to the addon's UUID via the API,
// mirroring migrateAddonAttachmentStateV0toV1.
func (r *addonAttachmentResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":        schema.StringAttribute{Computed: true},
					"app_id":    schema.StringAttribute{Optional: true, Computed: true},
					"addon_id":  schema.StringAttribute{Optional: true, Computed: true},
					"name":      schema.StringAttribute{Optional: true, Computed: true},
					"namespace": schema.StringAttribute{Optional: true, Computed: true},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior addonAttachmentResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Resolve the addon UUID from the stored app_id + id, mirroring
				// migrateAddonAttachmentStateV0toV1. State upgrades can run before
				// the provider is configured, so guard on the API client being set.
				if r.config != nil && !prior.AppID.IsNull() && !prior.ID.IsNull() {
					addon, err := r.config.Api.AddOnInfoByApp(ctx, prior.AppID.ValueString(), prior.ID.ValueString())
					if err != nil {
						resp.Diagnostics.AddError(
							"Error upgrading heroku_addon_attachment state (v0 to v1)",
							fmt.Sprintf("Could not resolve addon for attachment %s: %s", prior.ID.ValueString(), err),
						)
						return
					}
					if addon.ID != prior.ID.ValueString() {
						prior.AddonID = types.StringValue(addon.ID)
					}
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &prior)...)
			},
		},
	}
}

// readAddonAttachment fetches the current attachment state and populates m.
func (r *addonAttachmentResource) readAddonAttachment(ctx context.Context, m *addonAttachmentResourceModel) error {
	a, err := r.config.Api.AddOnAttachmentInfo(ctx, m.ID.ValueString())
	if err != nil {
		return fmt.Errorf("error retrieving addon attachment: %w", err)
	}

	m.AppID = types.StringValue(a.App.ID)
	m.AddonID = types.StringValue(a.Addon.ID)
	m.Name = types.StringValue(a.Name)

	if a.Namespace != nil {
		m.Namespace = types.StringValue(*a.Namespace)
	} else {
		m.Namespace = types.StringValue("")
	}

	return nil
}
