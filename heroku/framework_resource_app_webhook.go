package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*appWebhookResource)(nil)
	_ resource.ResourceWithConfigure   = (*appWebhookResource)(nil)
	_ resource.ResourceWithImportState = (*appWebhookResource)(nil)
)

// NewAppWebhookResource returns the framework implementation of the
// heroku_app_webhook resource.
func NewAppWebhookResource() resource.Resource {
	return &appWebhookResource{}
}

type appWebhookResource struct {
	config *Config
}

type appWebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	AppID         types.String `tfsdk:"app_id"`
	Level         types.String `tfsdk:"level"`
	URL           types.String `tfsdk:"url"`
	Include       types.List   `tfsdk:"include"`
	Secret        types.String `tfsdk:"secret"`
	Authorization types.String `tfsdk:"authorization"`
}

func (r *appWebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_webhook"
}

func (r *appWebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"level": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Required: true,
			},
			"include": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
			"secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"authorization": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (r *appWebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *appWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	include, diags := listToStringSlice(ctx, plan.Include)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.AppWebhookCreateOpts{
		Level:   plan.Level.ValueString(),
		URL:     plan.URL.ValueString(),
		Include: include,
	}

	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		s := plan.Secret.ValueString()
		opts.Secret = &s
	}

	if !plan.Authorization.IsNull() && !plan.Authorization.IsUnknown() {
		a := plan.Authorization.ValueString()
		opts.Authorization = &a
	}

	webhook, err := r.config.Api.AppWebhookCreate(ctx, plan.AppID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating app webhook", err.Error())
		return
	}

	plan.ID = types.StringValue(webhook.ID)
	r.setModelFromWebhook(ctx, &plan, webhook.ID, webhook.Include, webhook.Level, webhook.URL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.config.Api.AppWebhookInfo(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading app webhook", err.Error())
		return
	}

	r.setModelFromWebhook(ctx, &state, webhook.ID, webhook.Include, webhook.Level, webhook.URL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *appWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appWebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state appWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	plan.ID = state.ID

	opts := heroku.AppWebhookUpdateOpts{}

	if !plan.Level.Equal(state.Level) {
		v := plan.Level.ValueString()
		opts.Level = &v
	}

	if !plan.URL.Equal(state.URL) {
		v := plan.URL.ValueString()
		opts.URL = &v
	}

	if !plan.Include.Equal(state.Include) {
		include, diags := listToStringPointers(ctx, plan.Include)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts.Include = include
	}

	if !plan.Secret.Equal(state.Secret) {
		if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
			s := plan.Secret.ValueString()
			opts.Secret = &s
		} else {
			opts.Secret = nil
		}
	}

	if !plan.Authorization.Equal(state.Authorization) {
		if !plan.Authorization.IsNull() && !plan.Authorization.IsUnknown() {
			a := plan.Authorization.ValueString()
			opts.Authorization = &a
		} else {
			opts.Authorization = nil
		}
	}

	webhook, err := r.config.Api.AppWebhookUpdate(ctx, plan.AppID.ValueString(), plan.ID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating app webhook", err.Error())
		return
	}

	r.setModelFromWebhook(ctx, &plan, webhook.ID, webhook.Include, webhook.Level, webhook.URL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appWebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.config.Api.AppWebhookDelete(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting app webhook", err.Error())
		return
	}
}

func (r *appWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, webhookID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing app webhook", err.Error())
		return
	}

	webhook, err := r.config.Api.AppWebhookInfo(ctx, app, webhookID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading app webhook during import", err.Error())
		return
	}

	includeList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Include)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), webhook.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), webhook.App.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url"), webhook.URL)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("level"), webhook.Level)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("include"), includeList)...)
}

// setModelFromWebhook populates the model fields that are returned by the API.
// secret and authorization are intentionally omitted — the API never returns them.
func (r *appWebhookResource) setModelFromWebhook(ctx context.Context, m *appWebhookResourceModel, id string, include []string, level, url string) {
	m.ID = types.StringValue(id)
	m.Level = types.StringValue(level)
	m.URL = types.StringValue(url)

	includeList, diags := types.ListValueFrom(ctx, types.StringType, include)
	if !diags.HasError() {
		m.Include = includeList
	}
}

// listToStringSlice converts a types.List of strings to []string.
func listToStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

// listToStringPointers converts a types.List of strings to []*string,
// as required by AppWebhookUpdateOpts.Include.
func listToStringPointers(ctx context.Context, list types.List) ([]*string, diag.Diagnostics) {
	var strs []string
	diags := list.ElementsAs(ctx, &strs, false)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]*string, len(strs))
	for i := range strs {
		s := strs[i]
		out[i] = &s
	}
	return out, diags
}
