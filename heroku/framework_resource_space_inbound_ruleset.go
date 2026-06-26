package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource                   = (*spaceInboundRulesetResource)(nil)
	_ resource.ResourceWithConfigure      = (*spaceInboundRulesetResource)(nil)
	_ resource.ResourceWithValidateConfig = (*spaceInboundRulesetResource)(nil)
)

// NewSpaceInboundRulesetResource returns the framework implementation of the
// heroku_space_inbound_ruleset resource.
func NewSpaceInboundRulesetResource() resource.Resource {
	return &spaceInboundRulesetResource{}
}

type spaceInboundRulesetResource struct {
	config *Config
}

type spaceInboundRuleModel struct {
	Action types.String `tfsdk:"action"`
	Source types.String `tfsdk:"source"`
}

type spaceInboundRulesetResourceModel struct {
	ID    types.String            `tfsdk:"id"`
	Space types.String            `tfsdk:"space"`
	Rule  []spaceInboundRuleModel `tfsdk:"rule"`
}

func (r *spaceInboundRulesetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space_inbound_ruleset"
}

func (r *spaceInboundRulesetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		},
		Blocks: map[string]schema.Block{
			"rule": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Required: true,
						},
						"source": schema.StringAttribute{
							Required: true,
							// Restore SDKv2 validation.IsCIDRNetwork(0, 32) on source.
							Validators: []validator.String{
								cidrNetworkValidator(0, 32),
							},
						},
					},
				},
				// Restore SDKv2 MinItems: 1 on the rule block.
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

// ValidateConfig enforces the SDKv2 MinItems: 1 constraint on the rule block.
// An attribute/block validator (setvalidator.SizeAtLeast) cannot catch the
// zero-blocks case because an unconfigured block collection is null, and
// framework validators skip null values. This config-level check rejects a
// ruleset with no rule blocks at plan time, mirroring SDKv2.
func (r *spaceInboundRulesetResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data spaceInboundRulesetResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.Rule) < 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("rule"),
			"Missing rule block",
			"the rule set must contain at least 1 rule block",
		)
	}
}

func (r *spaceInboundRulesetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *spaceInboundRulesetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan spaceInboundRulesetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyRuleset(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating inbound ruleset", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *spaceInboundRulesetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state spaceInboundRulesetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleset, err := r.config.Api.InboundRulesetCurrent(ctx, state.Space.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading inbound ruleset",
			fmt.Sprintf("Error reading inbound ruleset for space (%s): %s", state.Space.ValueString(), err),
		)
		return
	}

	state.ID = types.StringValue(ruleset.ID)
	state.Rule = rulesFromAPI(ruleset.Rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *spaceInboundRulesetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan spaceInboundRulesetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state spaceInboundRulesetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	plan.ID = state.ID

	if err := r.applyRuleset(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating inbound ruleset", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete resets the inbound ruleset to Heroku's default (allow all traffic from
// 0.0.0.0/0). A Private Space must always have an inbound ruleset; there is no
// delete API method, so we restore the factory default instead.
func (r *spaceInboundRulesetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state spaceInboundRulesetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spaceIdentity := state.Space.ValueString()

	rules := []*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}{
		{Action: "allow", Source: "0.0.0.0/0"},
	}

	_, err := r.config.Api.InboundRulesetCreate(ctx, spaceIdentity, heroku.InboundRulesetCreateOpts{Rules: rules})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error resetting inbound ruleset",
			fmt.Sprintf("Error resetting inbound ruleset for space (%s): %s", spaceIdentity, err),
		)
		return
	}
}

// applyRuleset calls InboundRulesetCreate (a PUT upsert) for the given plan,
// then reads current state back into the model — mirroring the SDKv2 pattern
// of calling resourceHerokuSpaceInboundRulesetRead after the write.
func (r *spaceInboundRulesetResource) applyRuleset(ctx context.Context, m *spaceInboundRulesetResourceModel) error {
	spaceIdentity := m.Space.ValueString()

	var rules []*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}
	for _, rule := range m.Rule {
		entry := &struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}{
			Action: rule.Action.ValueString(),
			Source: rule.Source.ValueString(),
		}
		rules = append(rules, entry)
	}

	_, err := r.config.Api.InboundRulesetCreate(ctx, spaceIdentity, heroku.InboundRulesetCreateOpts{Rules: rules})
	if err != nil {
		return fmt.Errorf("error creating inbound ruleset for space (%s): %w", spaceIdentity, err)
	}

	// Read back the current ruleset to populate computed fields (id).
	ruleset, err := r.config.Api.InboundRulesetCurrent(ctx, spaceIdentity)
	if err != nil {
		return fmt.Errorf("error reading inbound ruleset for space (%s): %w", spaceIdentity, err)
	}

	m.ID = types.StringValue(ruleset.ID)
	m.Rule = rulesFromAPI(ruleset.Rules)
	return nil
}

// rulesFromAPI converts the API rule slice to the framework model slice.
func rulesFromAPI(apiRules []struct {
	Action string `json:"action" url:"action,key"`
	Source string `json:"source" url:"source,key"`
}) []spaceInboundRuleModel {
	out := make([]spaceInboundRuleModel, 0, len(apiRules))
	for _, r := range apiRules {
		out = append(out, spaceInboundRuleModel{
			Action: types.StringValue(r.Action),
			Source: types.StringValue(r.Source),
		})
	}
	return out
}
