// Pipeline Promotion Resource
//
// This resource allows promoting releases between apps in a Heroku Pipeline.
// Supports promoting either the latest release or a specific release by ID.
package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

var (
	_ resource.Resource              = (*pipelinePromotionResource)(nil)
	_ resource.ResourceWithConfigure = (*pipelinePromotionResource)(nil)
)

// NewPipelinePromotionResource returns the framework implementation of the
// heroku_pipeline_promotion resource.
func NewPipelinePromotionResource() resource.Resource {
	return &pipelinePromotionResource{}
}

type pipelinePromotionResource struct {
	config *Config
}

// promotedReleaseModel holds a single entry from the promoted_release_ids block.
type promotedReleaseModel struct {
	AppID     types.String `tfsdk:"app_id"`
	ReleaseID types.String `tfsdk:"release_id"`
}

type pipelinePromotionResourceModel struct {
	ID                 types.String           `tfsdk:"id"`
	Pipeline           types.String           `tfsdk:"pipeline"`
	SourceAppID        types.String           `tfsdk:"source_app_id"`
	ReleaseID          types.String           `tfsdk:"release_id"`
	Targets            types.Set              `tfsdk:"targets"`
	Status             types.String           `tfsdk:"status"`
	CreatedAt          types.String           `tfsdk:"created_at"`
	PromotedReleaseID  types.String           `tfsdk:"promoted_release_id"`
	PromotedReleaseIDs []promotedReleaseModel `tfsdk:"promoted_release_ids"`
}

func (r *pipelinePromotionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_promotion"
}

func (r *pipelinePromotionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline": schema.StringAttribute{
				Required:    true,
				Description: "Pipeline ID for the promotion",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 validation.IsUUID on pipeline.
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"source_app_id": schema.StringAttribute{
				Required:    true,
				Description: "Source app ID to promote from",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 validation.IsUUID on source_app_id.
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"release_id": schema.StringAttribute{
				Required:    true,
				Description: "Release ID to promote to target apps",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 validation.IsUUID on release_id.
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"targets": schema.SetAttribute{
				Required:    true,
				Description: "Set of target app IDs to promote to",
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				// Restore SDKv2 per-element validation.IsUUID on targets.
				Validators: []fwvalidator.Set{
					setvalidator.ValueStringsAre(uuidValidator()),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the promotion (pending, completed)",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the promotion was created",
			},
			"promoted_release_id": schema.StringAttribute{
				Computed:           true,
				DeprecationMessage: "Use promoted_release_ids instead. This attribute contains the first release ID from promoted_release_ids for backwards compatibility.",
				Description:        "ID of the first promoted release (deprecated, use promoted_release_ids)",
			},
		},
		Blocks: map[string]schema.Block{
			"promoted_release_ids": schema.ListNestedBlock{
				Description: "List of promoted releases with their associated target app IDs",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"app_id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the target app that received the promotion",
						},
						"release_id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the release created on the target app",
						},
					},
				},
			},
		},
	}
}

func (r *pipelinePromotionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pipelinePromotionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pipelinePromotionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[DEBUG] Creating pipeline promotion")

	pipelineID := plan.Pipeline.ValueString()
	sourceAppID := plan.SourceAppID.ValueString()
	releaseIDStr := plan.ReleaseID.ValueString()

	opts := heroku.PipelinePromotionCreateOpts{}
	opts.Pipeline.ID = pipelineID
	opts.Source.App = &struct {
		ID *string `json:"id,omitempty" url:"id,omitempty,key"`
	}{ID: &sourceAppID}
	opts.Source.Release = &struct {
		ID *string `json:"id,omitempty" url:"id,omitempty,key"`
	}{ID: &releaseIDStr}

	log.Printf("[DEBUG] Promoting release: %s", releaseIDStr)

	// Convert targets set to slice of target structs.
	var targetIDs []string
	resp.Diagnostics.Append(plan.Targets.ElementsAs(ctx, &targetIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i := range targetIDs {
		id := targetIDs[i]
		opts.Targets = append(opts.Targets, struct {
			App *struct {
				ID *string `json:"id,omitempty" url:"id,omitempty,key"`
			} `json:"app,omitempty" url:"app,omitempty,key"`
		}{App: &struct {
			ID *string `json:"id,omitempty" url:"id,omitempty,key"`
		}{ID: &id}})
	}

	log.Printf("[DEBUG] Pipeline promotion create configuration: %#v", opts)

	promotion, err := r.config.Api.PipelinePromotionCreate(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error creating pipeline promotion", err.Error())
		return
	}

	log.Printf("[INFO] Created pipeline promotion ID: %s", promotion.ID)
	plan.ID = types.StringValue(promotion.ID)

	if err := r.readPromotionIntoModel(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline promotion after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pipelinePromotionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pipelinePromotionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readPromotionIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline promotion", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not applicable for pipeline promotions — all inputs are ForceNew
// and the promotion itself is immutable. This method exists only to satisfy
// the resource.Resource interface; in practice it should never be called.
func (r *pipelinePromotionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state pipelinePromotionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readPromotionIntoModel(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading pipeline promotion", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete is a no-op: Heroku has no delete endpoint for pipeline promotions.
// The framework removes the resource from state after this method returns
// without error.
func (r *pipelinePromotionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	log.Printf("[INFO] There is no DELETE for pipeline promotion resource so this is a no-op. Promotion will be removed from state.")
}

// readPromotionIntoModel fetches the promotion and its targets from the Heroku
// API and populates m, mirroring the SDKv2 resourceHerokuPipelinePromotionRead
// logic exactly.
func (r *pipelinePromotionResource) readPromotionIntoModel(ctx context.Context, m *pipelinePromotionResourceModel) error {
	id := m.ID.ValueString()
	log.Printf("[DEBUG] Reading pipeline promotion: %s", id)

	promotion, err := r.config.Api.PipelinePromotionInfo(ctx, id)
	if err != nil {
		return fmt.Errorf("error retrieving pipeline promotion: %s", err)
	}

	m.Status = types.StringValue(promotion.Status)
	m.CreatedAt = types.StringValue(promotion.CreatedAt.String())
	m.Pipeline = types.StringValue(promotion.Pipeline.ID)
	m.SourceAppID = types.StringValue(promotion.Source.App.ID)

	// Fetch promotion targets to get the resulting release IDs.
	targets, err := r.config.Api.PipelinePromotionTargetList(ctx, id, nil)
	if err != nil {
		return fmt.Errorf("error retrieving pipeline promotion targets: %s", err)
	}

	var promotedReleases []promotedReleaseModel
	var firstReleaseID string
	for _, target := range targets {
		if target.Release != nil && target.Release.ID != "" {
			promotedReleases = append(promotedReleases, promotedReleaseModel{
				AppID:     types.StringValue(target.App.ID),
				ReleaseID: types.StringValue(target.Release.ID),
			})
			if firstReleaseID == "" {
				firstReleaseID = target.Release.ID
			}
			log.Printf("[DEBUG] Found promoted release ID: %s for app: %s", target.Release.ID, target.App.ID)
		}
	}

	if promotedReleases == nil {
		promotedReleases = []promotedReleaseModel{}
	}
	m.PromotedReleaseIDs = promotedReleases

	if firstReleaseID != "" {
		m.PromotedReleaseID = types.StringValue(firstReleaseID)
	} else {
		m.PromotedReleaseID = types.StringNull()
	}

	log.Printf("[DEBUG] Pipeline promotion read completed for: %s with %d promoted releases", id, len(promotedReleases))
	return nil
}
