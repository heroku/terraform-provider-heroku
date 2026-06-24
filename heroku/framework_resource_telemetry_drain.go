package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource              = (*telemetryDrainResource)(nil)
	_ resource.ResourceWithConfigure = (*telemetryDrainResource)(nil)
)

// NewTelemetryDrainResource returns the framework implementation of the
// heroku_telemetry_drain resource.
func NewTelemetryDrainResource() resource.Resource {
	return &telemetryDrainResource{}
}

type telemetryDrainResource struct {
	config *Config
}

type telemetryDrainResourceModel struct {
	ID           types.String `tfsdk:"id"`
	OwnerID      types.String `tfsdk:"owner_id"`
	OwnerType    types.String `tfsdk:"owner_type"`
	Endpoint     types.String `tfsdk:"endpoint"`
	ExporterType types.String `tfsdk:"exporter_type"`
	Signals      types.Set    `tfsdk:"signals"`
	Headers      types.Map    `tfsdk:"headers"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *telemetryDrainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_telemetry_drain"
}

func (r *telemetryDrainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the app or space that owns this telemetry drain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					uuidValidator(),
				},
			},
			"owner_type": schema.StringAttribute{
				Required:    true,
				Description: "Type of owner (app or space)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("app", "space"),
				},
			},
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "URI of your OpenTelemetry consumer",
			},
			"exporter_type": schema.StringAttribute{
				Required:    true,
				Description: "Transport type for OpenTelemetry consumer (otlphttp or otlp)",
				Validators: []validator.String{
					stringvalidator.OneOf("otlphttp", "otlp"),
				},
			},
			"signals": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "OpenTelemetry signals to send (traces, metrics, logs)",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf("traces", "metrics", "logs"),
					),
				},
			},
			"headers": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Headers to send to your OpenTelemetry consumer",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the telemetry drain was created",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the telemetry drain was last updated",
			},
		},
	}
}

func (r *telemetryDrainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *telemetryDrainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan telemetryDrainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that the owner supports OpenTelemetry drains (Fir generation only).
	if err := validateOwnerSupportsOtel(r.config.Api, plan.OwnerID.ValueString(), plan.OwnerType.ValueString()); err != nil {
		resp.Diagnostics.AddError("Owner does not support telemetry drains", err.Error())
		return
	}

	// Extract signals from the set.
	var signalStrings []string
	resp.Diagnostics.Append(plan.Signals.ElementsAs(ctx, &signalStrings, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract headers from the map.
	var headerStrings map[string]string
	resp.Diagnostics.Append(plan.Headers.ElementsAs(ctx, &headerStrings, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.TelemetryDrainCreateOpts{
		Owner: struct {
			ID   string `json:"id" url:"id,key"`
			Type string `json:"type" url:"type,key"`
		}{
			ID:   plan.OwnerID.ValueString(),
			Type: plan.OwnerType.ValueString(),
		},
		Exporter: struct {
			Endpoint string            `json:"endpoint" url:"endpoint,key"`
			Headers  map[string]string `json:"headers,omitempty" url:"headers,omitempty,key"`
			Type     string            `json:"type" url:"type,key"`
		}{
			Endpoint: plan.Endpoint.ValueString(),
			Type:     plan.ExporterType.ValueString(),
			Headers:  headerStrings,
		},
		Signals: signalStrings,
	}

	drain, err := r.config.Api.TelemetryDrainCreate(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating telemetry drain", err.Error())
		return
	}

	plan.ID = types.StringValue(drain.ID)

	resp.Diagnostics.Append(r.setModelFromDrain(&plan, drain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *telemetryDrainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state telemetryDrainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	drain, err := r.config.Api.TelemetryDrainInfo(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading telemetry drain", err.Error())
		return
	}

	resp.Diagnostics.Append(r.setModelFromDrain(&state, drain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *telemetryDrainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan telemetryDrainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	var state telemetryDrainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	// Extract headers.
	var headerStrings map[string]string
	resp.Diagnostics.Append(plan.Headers.ElementsAs(ctx, &headerStrings, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract signals.
	var signalStrings []string
	resp.Diagnostics.Append(plan.Signals.ElementsAs(ctx, &signalStrings, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	signalPtrs := make([]*string, len(signalStrings))
	for i := range signalStrings {
		s := signalStrings[i]
		signalPtrs[i] = &s
	}

	opts := heroku.TelemetryDrainUpdateOpts{
		Exporter: &struct {
			Endpoint string            `json:"endpoint" url:"endpoint,key"`
			Headers  map[string]string `json:"headers,omitempty" url:"headers,omitempty,key"`
			Type     string            `json:"type" url:"type,key"`
		}{
			Endpoint: plan.Endpoint.ValueString(),
			Type:     plan.ExporterType.ValueString(),
			Headers:  headerStrings,
		},
		Signals: signalPtrs,
	}

	drain, err := r.config.Api.TelemetryDrainUpdate(ctx, plan.ID.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating telemetry drain", err.Error())
		return
	}

	resp.Diagnostics.Append(r.setModelFromDrain(&plan, drain)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *telemetryDrainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state telemetryDrainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.config.Api.TelemetryDrainDelete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting telemetry drain", err.Error())
		return
	}
}

// setModelFromDrain populates the model from a TelemetryDrain API response.
func (r *telemetryDrainResource) setModelFromDrain(m *telemetryDrainResourceModel, drain *heroku.TelemetryDrain) diag.Diagnostics {
	var diags diag.Diagnostics

	m.OwnerID = types.StringValue(drain.Owner.ID)
	m.OwnerType = types.StringValue(drain.Owner.Type)
	m.Endpoint = types.StringValue(drain.Exporter.Endpoint)
	m.ExporterType = types.StringValue(drain.Exporter.Type)
	m.CreatedAt = types.StringValue(drain.CreatedAt.String())
	m.UpdatedAt = types.StringValue(drain.UpdatedAt.String())

	// Convert []string signals to types.Set.
	signalElems := make([]attr.Value, len(drain.Signals))
	for i, s := range drain.Signals {
		signalElems[i] = types.StringValue(s)
	}
	signalSet, d := types.SetValue(types.StringType, signalElems)
	diags.Append(d...)
	if !d.HasError() {
		m.Signals = signalSet
	}

	// Convert map[string]string headers to types.Map.
	headerElems := make(map[string]attr.Value, len(drain.Exporter.Headers))
	for k, v := range drain.Exporter.Headers {
		headerElems[k] = types.StringValue(v)
	}
	headerMap, d := types.MapValue(types.StringType, headerElems)
	diags.Append(d...)
	if !d.HasError() {
		m.Headers = headerMap
	}

	return diags
}
