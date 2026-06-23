package heroku

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                 = (*domainResource)(nil)
	_ resource.ResourceWithConfigure    = (*domainResource)(nil)
	_ resource.ResourceWithImportState  = (*domainResource)(nil)
	_ resource.ResourceWithUpgradeState = (*domainResource)(nil)
)

// NewDomainResource returns the framework implementation of the heroku_domain resource.
func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	config *Config
}

type domainResourceModel struct {
	ID            types.String `tfsdk:"id"`
	AppID         types.String `tfsdk:"app_id"`
	Hostname      types.String `tfsdk:"hostname"`
	CName         types.String `tfsdk:"cname"`
	SniEndpointID types.String `tfsdk:"sni_endpoint_id"`
}

// domainV0ResourceModel mirrors resourceHerokuDomainV0() for the state upgrader.
type domainV0ResourceModel struct {
	ID            types.String `tfsdk:"id"`
	App           types.String `tfsdk:"app"`
	Hostname      types.String `tfsdk:"hostname"`
	CName         types.String `tfsdk:"cname"`
	SniEndpointID types.String `tfsdk:"sni_endpoint_id"`
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"hostname": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cname": schema.StringAttribute{
				Computed: true,
			},
			"sni_endpoint_id": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()
	opts := heroku.DomainCreateOpts{
		Hostname: plan.Hostname.ValueString(),
	}

	if v := plan.SniEndpointID.ValueString(); v != "" {
		opts.SniEndpoint = &v
	}

	do, err := r.config.Api.DomainCreate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	if err := r.populateModel(ctx, &plan, do); err != nil {
		resp.Diagnostics.AddError("Error populating domain attributes", err.Error())
		return
	}

	time.Sleep(time.Duration(r.config.PostDomainCreateDelay) * time.Second)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := state.AppID.ValueString()
	do, err := r.config.Api.DomainInfo(ctx, appID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving domain", err.Error())
		return
	}

	if err := r.populateModel(ctx, &state, do); err != nil {
		resp.Diagnostics.AddError("Error populating domain attributes", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update handles in-place updates for sni_endpoint_id (all other writable fields are ForceNew).
func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	appID := plan.AppID.ValueString()
	sniVal := plan.SniEndpointID.ValueString()
	opts := heroku.DomainUpdateOpts{
		SniEndpoint: &sniVal,
	}

	do, err := r.config.Api.DomainUpdate(ctx, appID, plan.ID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	if err := r.populateModel(ctx, &plan, do); err != nil {
		resp.Diagnostics.AddError("Error populating domain attributes", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.config.Api.DomainDelete(ctx, state.AppID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting domain", fmt.Sprintf("Error deleting domain: %s", err))
		return
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, id, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing domain", err.Error())
		return
	}

	do, err := r.config.Api.DomainInfo(ctx, app, id)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving domain during import", err.Error())
		return
	}

	var state domainResourceModel
	if err := r.populateModel(ctx, &state, do); err != nil {
		resp.Diagnostics.AddError("Error populating domain attributes", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// UpgradeState implements resource.ResourceWithUpgradeState.
// The SDKv2 resource had SchemaVersion 1 with a single v0→v1 upgrader that
// replaced the "app" attribute (name or UUID) with "app_id" (UUID).
func (r *domainResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuDomainV0()
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"hostname": schema.StringAttribute{
				Required: true,
			},
			"cname": schema.StringAttribute{
				Computed: true,
			},
			"sni_endpoint_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior domainV0ResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading domain state", err.Error())
					return
				}

				upgraded := domainResourceModel{
					ID:            prior.ID,
					AppID:         types.StringValue(appID),
					Hostname:      prior.Hostname,
					CName:         prior.CName,
					SniEndpointID: prior.SniEndpointID,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// populateModel fills in a domainResourceModel from a Heroku Domain API response,
// mirroring the SDKv2 populateResource function.
// When ACM is active on the app, the sni_endpoint_id is not captured (matches SDKv2 behaviour).
func (r *domainResource) populateModel(_ context.Context, m *domainResourceModel, do *heroku.Domain) error {
	m.ID = types.StringValue(do.ID)
	m.AppID = types.StringValue(do.App.ID)
	m.Hostname = types.StringValue(do.Hostname)

	if do.CName != nil {
		m.CName = types.StringValue(*do.CName)
	} else {
		m.CName = types.StringNull()
	}

	// Do not capture SNI Endpoint when ACM is active (mirrors SDKv2 populateResource).
	hasACM, err := retrieveAcm(do.App.ID, r.config.Api)
	if err != nil {
		return fmt.Errorf("error checking ACM status: %w", err)
	}
	if !hasACM {
		if do.SniEndpoint != nil {
			m.SniEndpointID = types.StringValue(do.SniEndpoint.ID)
		} else {
			m.SniEndpointID = types.StringNull()
		}
	}
	// When ACM is active, leave m.SniEndpointID untouched — the SDKv2 code
	// likewise skips calling d.Set("sni_endpoint_id", ...) in that case, so the
	// prior state value is preserved.

	return nil
}
