package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*sslResource)(nil)
	_ resource.ResourceWithConfigure   = (*sslResource)(nil)
	_ resource.ResourceWithImportState = (*sslResource)(nil)
)

// NewSSLResource returns the framework implementation of the heroku_ssl resource.
func NewSSLResource() resource.Resource {
	return &sslResource{}
}

type sslResource struct {
	config *Config
}

type sslResourceModel struct {
	ID               types.String `tfsdk:"id"`
	AppID            types.String `tfsdk:"app_id"`
	CertificateChain types.String `tfsdk:"certificate_chain"`
	PrivateKey       types.String `tfsdk:"private_key"`
	Name             types.String `tfsdk:"name"`
}

func (r *sslResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssl"
}

func (r *sslResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"certificate_chain": schema.StringAttribute{
				Required: true,
			},
			"private_key": schema.StringAttribute{
				// Optional (not Required) to keep import easier, matching SDKv2 behavior.
				Optional:  true,
				Sensitive: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *sslResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sslResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sslResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()

	opts := heroku.SniEndpointCreateOpts{
		CertificateChain: plan.CertificateChain.ValueString(),
		PrivateKey:       plan.PrivateKey.ValueString(),
	}

	log.Printf("[DEBUG] Creating SSL certificate for app %#v", appID)

	ep, err := r.config.Api.SniEndpointCreate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SSL certificate",
			fmt.Sprintf("Error creating SSL certificate for app %s: %s", appID, err.Error()),
		)
		return
	}

	log.Printf("[DEBUG] Created SSL Certificate %s", ep.ID)

	plan.ID = types.StringValue(ep.ID)
	r.setModelFromEndpoint(&plan, ep)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sslResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sslResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep, err := r.config.Api.SniEndpointInfo(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading SSL certificate", err.Error())
		return
	}

	r.setModelFromEndpoint(&state, ep)

	// private_key is not returned by the API; preserve state value.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sslResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sslResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()

	opts := heroku.SniEndpointUpdateOpts{
		CertificateChain: plan.CertificateChain.ValueString(),
		PrivateKey:       plan.PrivateKey.ValueString(),
	}

	log.Printf("[DEBUG] Updating SSL Certificate configuration: %#v, %#v", appID, opts)

	ep, err := r.config.Api.SniEndpointUpdate(ctx, appID, plan.ID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SSL certificate", fmt.Sprintf("Error updating Sni endpoint: %s", err))
		return
	}

	log.Printf("[DEBUG] Updated SSL Certificate configuration: %#v, %#v", appID, opts)

	r.setModelFromEndpoint(&plan, ep)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sslResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sslResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting SSL Cert: %s", state.ID.ValueString())

	_, err := r.config.Api.SniEndpointDelete(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting SSL certificate", fmt.Sprintf("Error deleting SSL Cert: %s", err))
		return
	}

	log.Printf("[INFO] Deleted SSL Cert: %s", state.ID.ValueString())
}

func (r *sslResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, certID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing SSL certificate", err.Error())
		return
	}

	ep, err := r.config.Api.SniEndpointInfo(ctx, app, certID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading SSL certificate during import", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ep.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), ep.App.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("certificate_chain"), ep.CertificateChain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), ep.Name)...)
	// private_key is intentionally omitted from import (not returned by API).
}

func (r *sslResource) setModelFromEndpoint(m *sslResourceModel, ep *heroku.SniEndpoint) {
	m.AppID = types.StringValue(ep.App.ID)
	m.CertificateChain = types.StringValue(ep.CertificateChain)
	m.Name = types.StringValue(ep.Name)
	// private_key is not returned by the Heroku API; leave m.PrivateKey unchanged.
}
