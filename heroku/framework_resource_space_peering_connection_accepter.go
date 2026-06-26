package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*spacePeeringConnectionAccepterResource)(nil)
	_ resource.ResourceWithConfigure   = (*spacePeeringConnectionAccepterResource)(nil)
	_ resource.ResourceWithImportState = (*spacePeeringConnectionAccepterResource)(nil)
)

// NewSpacePeeringConnectionAccepterResource returns the framework implementation
// of the heroku_space_peering_connection_accepter resource.
func NewSpacePeeringConnectionAccepterResource() resource.Resource {
	return &spacePeeringConnectionAccepterResource{}
}

type spacePeeringConnectionAccepterResource struct {
	config *Config
}

type spacePeeringConnectionAccepterResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Space                  types.String `tfsdk:"space"`
	VpcPeeringConnectionID types.String `tfsdk:"vpc_peering_connection_id"`
	Status                 types.String `tfsdk:"status"`
	Type                   types.String `tfsdk:"type"`
}

func (r *spacePeeringConnectionAccepterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_peering_connection_accepter"
}

func (r *spacePeeringConnectionAccepterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"vpc_peering_connection_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *spacePeeringConnectionAccepterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *spacePeeringConnectionAccepterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan spacePeeringConnectionAccepterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spaceIdentity := plan.Space.ValueString()
	pcxID := plan.VpcPeeringConnectionID.ValueString()

	// There is a lag between when a peering request is initiated from AWS's end
	// and when it appears as an option in a space's list of peering connections.
	// In testing, this is usually in the 1-3 minute range. We retry for 5 minutes
	// so plan/apply runs that create the two resources at the same time don't
	// result in an error.
	acceptDeadline := time.Now().Add(5 * time.Minute)
	var acceptErr error
	for {
		_, acceptErr = r.config.Api.PeeringAccept(ctx, spaceIdentity, pcxID)
		if acceptErr == nil {
			log.Printf("[INFO] Peer connection %s to %s has been accepted", pcxID, spaceIdentity)
			break
		}
		if time.Now().After(acceptDeadline) {
			break
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Context cancelled while accepting peer connection", ctx.Err().Error())
			return
		case <-time.After(10 * time.Second):
		}
	}
	if acceptErr != nil {
		resp.Diagnostics.AddError(
			"Error accepting peer connection",
			fmt.Sprintf("Unable to accept peer connection %s to %s: %s", pcxID, spaceIdentity, acceptErr),
		)
		return
	}

	log.Printf("[INFO] Space ID: %s, Peering Connection ID: %s", spaceIdentity, pcxID)

	plan.ID = types.StringValue(pcxID)

	log.Printf("[DEBUG] Waiting for connection (%s) to be accepted", pcxID)

	// Poll until the peering connection reaches the "active" state (up to 20 min).
	pending := map[string]bool{
		"initiating-request": true,
		"pending":            true,
		"pending-acceptance": true,
		"provisioning":       true,
	}
	waitDeadline := time.Now().Add(20 * time.Minute)
	for {
		p, err := r.config.Api.PeeringInfo(ctx, spaceIdentity, pcxID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for space peering connection",
				fmt.Sprintf("Error reading peering connection %s: %s", pcxID, err),
			)
			return
		}
		if p.Status == "active" {
			plan.Status = types.StringValue(p.Status)
			plan.Type = types.StringValue(p.Type)
			break
		}
		if !pending[p.Status] {
			resp.Diagnostics.AddError(
				"Unexpected peering connection status",
				fmt.Sprintf("Space peering connection %s reached unexpected status %q", pcxID, p.Status),
			)
			return
		}
		if time.Now().After(waitDeadline) {
			resp.Diagnostics.AddError(
				"Timeout waiting for space peering connection",
				fmt.Sprintf("Space peering connection %s did not become active within 20 minutes (last status: %s)", pcxID, p.Status),
			)
			return
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Context cancelled while waiting for peer connection", ctx.Err().Error())
			return
		case <-time.After(15 * time.Second):
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *spacePeeringConnectionAccepterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state spacePeeringConnectionAccepterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peeringConn, err := r.config.Api.PeeringInfo(ctx, state.Space.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading space peering connection", err.Error())
		return
	}

	state.ID = types.StringValue(peeringConn.PcxID)
	state.VpcPeeringConnectionID = types.StringValue(peeringConn.PcxID)
	state.Status = types.StringValue(peeringConn.Status)
	state.Type = types.StringValue(peeringConn.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the framework interface. All mutable attributes are
// ForceNew, so this method is never called in practice.
func (r *spacePeeringConnectionAccepterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes are either Computed-only or ForceNew; updates are not possible.
	// Re-read current state and write it back so state remains consistent.
	var state spacePeeringConnectionAccepterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *spacePeeringConnectionAccepterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state spacePeeringConnectionAccepterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting space peering connection: %s", state.ID.ValueString())

	_, err := r.config.Api.PeeringDestroy(ctx, state.Space.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting space peering connection", err.Error())
		return
	}
}

func (r *spacePeeringConnectionAccepterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	spaceIdentity, peeringPcxID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing space peering connection accepter", err.Error())
		return
	}

	peeringConn, err := r.config.Api.PeeringInfo(ctx, spaceIdentity, peeringPcxID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading space peering connection during import", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), peeringConn.PcxID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("space"), spaceIdentity)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vpc_peering_connection_id"), peeringConn.PcxID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status"), peeringConn.Status)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), peeringConn.Type)...)
}
