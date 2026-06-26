package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                = (*spaceVPNConnectionResource)(nil)
	_ resource.ResourceWithConfigure   = (*spaceVPNConnectionResource)(nil)
	_ resource.ResourceWithImportState = (*spaceVPNConnectionResource)(nil)
)

// NewSpaceVPNConnectionResource returns the framework implementation of the
// heroku_space_vpn_connection resource.
func NewSpaceVPNConnectionResource() resource.Resource {
	return &spaceVPNConnectionResource{}
}

type spaceVPNConnectionResource struct {
	config *Config
}

// spaceVPNConnectionTunnelModel represents a single tunnel entry.
type spaceVPNConnectionTunnelModel struct {
	IP           types.String `tfsdk:"ip"`
	PreSharedKey types.String `tfsdk:"pre_shared_key"`
}

type spaceVPNConnectionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Space          types.String `tfsdk:"space"`
	Name           types.String `tfsdk:"name"`
	PublicIP       types.String `tfsdk:"public_ip"`
	RoutableCidrs  types.Set    `tfsdk:"routable_cidrs"`
	SpaceCIDRBlock types.String `tfsdk:"space_cidr_block"`
	IKEVersion     types.Int64  `tfsdk:"ike_version"`
	Tunnels        types.List   `tfsdk:"tunnels"`
}

// tunnelAttrTypes returns the attribute types for a tunnel object.
func tunnelAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip":             types.StringType,
		"pre_shared_key": types.StringType,
	}
}

func (r *spaceVPNConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_vpn_connection"
}

func (r *spaceVPNConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_ip": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"routable_cidrs": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"space_cidr_block": schema.StringAttribute{
				Computed: true,
			},
			"ike_version": schema.Int64Attribute{
				Computed: true,
			},
			"tunnels": schema.ListNestedAttribute{
				Computed: true,
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Computed: true,
							Optional: true,
						},
						"pre_shared_key": schema.StringAttribute{
							Computed:  true,
							Optional:  true,
							Sensitive: true,
						},
					},
				},
			},
		},
	}
}

func (r *spaceVPNConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *spaceVPNConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan spaceVPNConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	space := plan.Space.ValueString()

	var routableCIDRs []string
	resp.Diagnostics.Append(plan.RoutableCidrs.ElementsAs(ctx, &routableCIDRs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.config.Api.VPNConnectionCreate(ctx, space, heroku.VPNConnectionCreateOpts{
		Name:          plan.Name.ValueString(),
		PublicIP:      plan.PublicIP.ValueString(),
		RoutableCidrs: routableCIDRs,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating VPN connection", fmt.Sprintf("Error creating VPN: %v", err))
		return
	}

	log.Printf("[DEBUG] Waiting for VPN (%s) to be allocated", conn.ID)

	// Poll until the VPN connection reaches "active" state (up to 30 minutes),
	// matching the SDKv2 StateChangeConf timeout of 30 minutes with 20-second intervals.
	waitDeadline := time.Now().Add(30 * time.Minute)
	for {
		vpn, vpnErr := r.config.Api.VPNConnectionInfo(ctx, space, conn.ID)

		// Retry on "VPN is not found" — the resource may not be visible immediately.
		if vpnErr != nil && strings.Contains(vpnErr.Error(), "VPN is not found") {
			if time.Now().After(waitDeadline) {
				resp.Diagnostics.AddError(
					"Timeout waiting for VPN connection",
					fmt.Sprintf("VPN connection %s did not become active within 30 minutes", conn.ID),
				)
				return
			}
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Context cancelled while waiting for VPN connection", ctx.Err().Error())
				return
			case <-time.After(20 * time.Second):
			}
			continue
		}

		if vpnErr != nil {
			resp.Diagnostics.AddError("Error waiting for VPN connection", fmt.Sprintf("error fetching VPN connection status: %s", vpnErr))
			return
		}

		if vpn.Status == "active" {
			break
		}

		if time.Now().After(waitDeadline) {
			resp.Diagnostics.AddError(
				"Timeout waiting for VPN connection",
				fmt.Sprintf("VPN connection %s did not become active within 30 minutes (last status: %s)", conn.ID, vpn.Status),
			)
			return
		}

		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Context cancelled while waiting for VPN connection", ctx.Err().Error())
			return
		case <-time.After(20 * time.Second):
		}
	}

	plan.ID = types.StringValue(buildCompositeID(space, conn.ID))

	if err := r.readVPNConnection(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading VPN connection after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *spaceVPNConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state spaceVPNConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readVPNConnection(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading VPN connection", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the framework interface. All mutable attributes are
// ForceNew (RequiresReplace), so this method is never called in practice.
func (r *spaceVPNConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state spaceVPNConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *spaceVPNConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state spaceVPNConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	space, id, err := parseCompositeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing VPN connection ID", err.Error())
		return
	}

	if _, err := r.config.Api.VPNConnectionDestroy(ctx, space, id); err != nil {
		resp.Diagnostics.AddError("Error deleting VPN connection", fmt.Sprintf("Error deleting VPN: %v", err))
		return
	}
}

func (r *spaceVPNConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the composite ID "space:vpn_connection_uuid", matching the
	// resource's own ID format (same as SDKv2 ImportStatePassthrough behavior).
	space, id, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing VPN connection", err.Error())
		return
	}

	// Pre-populate enough state for the subsequent Read refresh to succeed.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("space"), space)...)

	// Fetch the connection now so we can set name (required, non-computed).
	conn, err := r.config.Api.VPNConnectionInfo(ctx, space, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading VPN connection during import", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), conn.Name)...)
}

// readVPNConnection calls VPNConnectionInfo and populates the model.
func (r *spaceVPNConnectionResource) readVPNConnection(ctx context.Context, m *spaceVPNConnectionResourceModel) error {
	space, id, err := parseCompositeID(m.ID.ValueString())
	if err != nil {
		return err
	}

	conn, err := r.config.Api.VPNConnectionInfo(ctx, space, id)
	if err != nil {
		return fmt.Errorf("Error reading VPN information: %v", err)
	}

	m.Space = types.StringValue(space)
	m.Name = types.StringValue(conn.Name)
	m.PublicIP = types.StringValue(conn.PublicIP)
	m.SpaceCIDRBlock = types.StringValue(conn.SpaceCIDRBlock)
	m.IKEVersion = types.Int64Value(int64(conn.IKEVersion))

	cidrs, diags := types.SetValueFrom(ctx, types.StringType, conn.RoutableCidrs)
	if diags.HasError() {
		return fmt.Errorf("error converting routable_cidrs: %v", diags)
	}
	m.RoutableCidrs = cidrs

	tunnelObjs := make([]attr.Value, 0, len(conn.Tunnels))
	for _, t := range conn.Tunnels {
		obj, diags := types.ObjectValue(tunnelAttrTypes(), map[string]attr.Value{
			"ip":             types.StringValue(t.IP),
			"pre_shared_key": types.StringValue(t.PreSharedKey),
		})
		if diags.HasError() {
			return fmt.Errorf("error constructing tunnel object: %v", diags)
		}
		tunnelObjs = append(tunnelObjs, obj)
	}
	tunnelList, diags := types.ListValue(types.ObjectType{AttrTypes: tunnelAttrTypes()}, tunnelObjs)
	if diags.HasError() {
		return fmt.Errorf("error constructing tunnels list: %v", diags)
	}
	m.Tunnels = tunnelList

	return nil
}
