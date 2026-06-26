package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*spaceResource)(nil)
	_ resource.ResourceWithConfigure   = (*spaceResource)(nil)
	_ resource.ResourceWithImportState = (*spaceResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*spaceResource)(nil)
)

// NewSpaceResource returns the framework implementation of the heroku_space resource.
func NewSpaceResource() resource.Resource {
	return &spaceResource{}
}

type spaceResource struct {
	config *Config
}

type spaceResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Organization types.String `tfsdk:"organization"`
	CIDR         types.String `tfsdk:"cidr"`
	DataCIDR     types.String `tfsdk:"data_cidr"`
	OutboundIPs  types.List   `tfsdk:"outbound_ips"`
	Region       types.String `tfsdk:"region"`
	Shield       types.Bool   `tfsdk:"shield"`
	Generation   types.String `tfsdk:"generation"`
}

// staticStringDefault is an inline defaults.String implementation for a static
// string value. There is no "stringdefault" sub-package in this vendored version
// of terraform-plugin-framework, so we implement it here.
type staticStringDefault struct {
	value string
}

func (d staticStringDefault) Description(_ context.Context) string {
	return fmt.Sprintf("value defaults to %q", d.value)
}

func (d staticStringDefault) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value defaults to `%q`", d.value)
}

func (d staticStringDefault) DefaultString(_ context.Context, _ defaults.StringRequest, resp *defaults.StringResponse) {
	resp.PlanValue = types.StringValue(d.value)
}

// spaceBoolRequiresReplace is a planmodifier.Bool that marks the resource for
// replacement whenever the bool value changes. This mirrors the SDKv2 ForceNew
// behaviour for bool attributes. There is no "boolplanmodifier" sub-package in
// this vendored version of terraform-plugin-framework.
type spaceBoolRequiresReplace struct{}

func (m spaceBoolRequiresReplace) Description(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m spaceBoolRequiresReplace) MarkdownDescription(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m spaceBoolRequiresReplace) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	// Skip on create (no prior state).
	if req.StateValue.IsNull() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// spaceListUseStateForUnknown is a planmodifier.List that keeps the prior state
// value for a computed list when the plan value is unknown. There is no
// "listplanmodifier" sub-package in this vendored version of
// terraform-plugin-framework.
type spaceListUseStateForUnknown struct{}

func (m spaceListUseStateForUnknown) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

func (m spaceListUseStateForUnknown) MarkdownDescription(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

func (m spaceListUseStateForUnknown) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

func (r *spaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space"
}

func (r *spaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			},
			"organization": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"data_cidr": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"outbound_ips": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					spaceListUseStateForUnknown{},
				},
			},
			"region": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"shield": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					spaceBoolRequiresReplace{},
				},
			},
			"generation": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     staticStringDefault{value: "cedar"},
				Description: "Generation of the space. Defaults to cedar for backward compatibility.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 StringInSlice([]string{"cedar", "fir"}).
				Validators: []fwvalidator.String{
					stringvalidator.OneOf("cedar", "fir"),
				},
			},
		},
	}
}

func (r *spaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan ports resourceHerokuSpaceCustomizeDiff: if generation and shield
// are both known/set and shield==true and !IsFeatureSupported(generation,"space","shield"),
// add a plan error. Does nothing on destroy (req.Plan is null).
func (r *spaceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy plan — nothing to validate.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan spaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only validate when both generation and shield are known.
	if plan.Generation.IsNull() || plan.Generation.IsUnknown() {
		return
	}
	if plan.Shield.IsNull() || plan.Shield.IsUnknown() {
		return
	}

	generationStr := plan.Generation.ValueString()
	shieldBool := plan.Shield.ValueBool()

	if shieldBool && !IsFeatureSupported(generationStr, "space", "shield") {
		resp.Diagnostics.AddError(
			"Unsupported shield configuration",
			fmt.Sprintf("shield spaces are not supported for %s generation", generationStr),
		)
	}
}

func (r *spaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan spaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := heroku.SpaceCreateOpts{
		Name: plan.Name.ValueString(),
		Team: plan.Organization.ValueString(),
	}

	if !plan.Region.IsNull() && !plan.Region.IsUnknown() && plan.Region.ValueString() != "" {
		region := plan.Region.ValueString()
		opts.Region = &region
	}

	if !plan.Shield.IsNull() && !plan.Shield.IsUnknown() {
		shield := plan.Shield.ValueBool()
		if shield {
			log.Printf("[DEBUG] Creating a shield space")
		}
		opts.Shield = &shield
	}

	if !plan.CIDR.IsNull() && !plan.CIDR.IsUnknown() && plan.CIDR.ValueString() != "" {
		cidr := plan.CIDR.ValueString()
		opts.CIDR = &cidr
	}

	if !plan.DataCIDR.IsNull() && !plan.DataCIDR.IsUnknown() && plan.DataCIDR.ValueString() != "" {
		dataCIDR := plan.DataCIDR.ValueString()
		opts.DataCIDR = &dataCIDR
	}

	if !plan.Generation.IsNull() && !plan.Generation.IsUnknown() && plan.Generation.ValueString() != "" {
		generation := plan.Generation.ValueString()
		log.Printf("[DEBUG] Creating space with generation: %s", generation)
		opts.Generation = &generation
	}

	space, err := r.config.Api.SpaceCreate(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating space", err.Error())
		return
	}

	plan.ID = types.StringValue(space.ID)
	log.Printf("[INFO] Space ID: %s", space.ID)

	// Wait for the Space to be allocated, mirroring SpaceStateRefreshFunc with
	// pending "allocating" → target "allocated" and a 20-minute overall timeout.
	log.Printf("[DEBUG] Waiting for Space (%s) to be allocated", space.ID)
	waitDeadline := time.Now().Add(20 * time.Minute)
	for {
		spaceInfo, pollErr := r.config.Api.SpaceInfo(ctx, space.ID)
		if pollErr != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error polling space %s during allocation", space.ID),
				pollErr.Error(),
			)
			return
		}

		if spaceInfo.State == "allocated" {
			log.Printf("[DEBUG] Space %s is now allocated", space.ID)
			break
		}

		log.Printf("[DEBUG] Still allocating: %s (%s)", spaceInfo.State, space.ID)

		if time.Now().After(waitDeadline) {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Timeout waiting for Space (%s) to become available", space.ID),
				fmt.Sprintf("Space did not reach 'allocated' state within 20 minutes (last state: %s)", spaceInfo.State),
			)
			return
		}

		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Context cancelled while waiting for space allocation", ctx.Err().Error())
			return
		case <-time.After(15 * time.Second):
		}
	}

	// Post-create delay from provider configuration, matching SDKv2 behavior.
	time.Sleep(time.Duration(r.config.PostSpaceCreateDelay) * time.Second)

	if err := r.readSpaceIntoModel(ctx, &plan); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading space after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *spaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state spaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readSpaceIntoModel(ctx, &state); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading space", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *spaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan spaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state spaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the ID from state.
	plan.ID = state.ID

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		opts := heroku.SpaceUpdateOpts{Name: &name}

		_, err := r.config.Api.SpaceUpdate(ctx, state.ID.ValueString(), opts)
		if err != nil {
			resp.Diagnostics.AddError("Error updating space", err.Error())
			return
		}
	}

	if err := r.readSpaceIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading space after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *spaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state spaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting space: %s", state.ID.ValueString())
	_, err := r.config.Api.SpaceDelete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting space", err.Error())
		return
	}
}

func (r *spaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// heroku_space is imported by its ID alone (same as SDKv2 ImportStatePassthrough).
	state := spaceResourceModel{
		ID: types.StringValue(req.ID),
	}

	if err := r.readSpaceIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading space during import", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// readSpaceIntoModel fetches current space state via SpaceStateRefreshFunc
// (which internally calls SpaceInfo + SpaceNATInfo) and populates the model,
// mirroring the SDKv2 resourceHerokuSpaceRead logic exactly.
func (r *spaceResource) readSpaceIntoModel(ctx context.Context, m *spaceResourceModel) error {
	spaceRaw, _, err := SpaceStateRefreshFunc(r.config.Api, m.ID.ValueString())()
	if err != nil {
		return err
	}

	space := spaceRaw.(*spaceWithNAT)

	m.Name = types.StringValue(space.Name)
	m.Organization = types.StringValue(space.Organization.Name)
	m.Region = types.StringValue(space.Region.Name)
	m.Shield = types.BoolValue(space.Shield)
	m.CIDR = types.StringValue(space.CIDR)
	m.DataCIDR = types.StringValue(space.DataCIDR)
	m.Generation = types.StringValue(space.Generation.Name)

	// Convert NAT sources ([]string) into a types.List of strings.
	outboundIPElems := make([]types.String, len(space.NAT.Sources))
	for i, ip := range space.NAT.Sources {
		outboundIPElems[i] = types.StringValue(ip)
	}
	outboundIPList, diags := types.ListValueFrom(ctx, types.StringType, outboundIPElems)
	if diags.HasError() {
		return fmt.Errorf("error building outbound_ips list: %s", diags)
	}
	m.OutboundIPs = outboundIPList

	log.Printf("[DEBUG] Set NAT source IPs to %s for %s", space.NAT.Sources, m.ID.ValueString())

	return nil
}
