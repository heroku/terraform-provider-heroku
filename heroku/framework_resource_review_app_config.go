package heroku

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

// Deploy target types for heroku_review_app_config. Preserved from the original
// SDKv2 resource_heroku_review_app_config.go.
const (
	DeployTargetTypeSpace  = "space"
	DeployTargetTypeRegion = "region"
)

var (
	_ resource.Resource                = (*reviewAppConfigResource)(nil)
	_ resource.ResourceWithConfigure   = (*reviewAppConfigResource)(nil)
	_ resource.ResourceWithImportState = (*reviewAppConfigResource)(nil)
)

// NewReviewAppConfigResource returns the framework implementation of the
// heroku_review_app_config resource.
func NewReviewAppConfigResource() resource.Resource {
	return &reviewAppConfigResource{}
}

type reviewAppConfigResource struct {
	config *Config
}

type reviewAppConfigDeployTargetModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

type reviewAppConfigResourceModel struct {
	ID                  types.String                       `tfsdk:"id"`
	PipelineID          types.String                       `tfsdk:"pipeline_id"`
	OrgRepo             types.String                       `tfsdk:"org_repo"`
	DeployTarget        []reviewAppConfigDeployTargetModel `tfsdk:"deploy_target"`
	AutomaticReviewApps types.Bool                         `tfsdk:"automatic_review_apps"`
	BaseName            types.String                       `tfsdk:"base_name"`
	DestroyStaleApps    types.Bool                         `tfsdk:"destroy_stale_apps"`
	StaleDays           types.Int64                        `tfsdk:"stale_days"`
	WaitForCI           types.Bool                         `tfsdk:"wait_for_ci"`
	RepoID              types.Int64                        `tfsdk:"repo_id"`
}

func (r *reviewAppConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_review_app_config"
}

func (r *reviewAppConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				// TODO: port validation (IsUUID)
			},
			"org_repo": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"automatic_review_apps": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"base_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"destroy_stale_apps": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"stale_days": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				// TODO: port validation (IntBetween(1, 30)) and RequiredWith destroy_stale_apps
			},
			"wait_for_ci": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"repo_id": schema.Int64Attribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"deploy_target": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required: true,
							// TODO: port validation (validateDeployTargetID: UUID or 2-letter region code)
						},
						"type": schema.StringAttribute{
							Required: true,
							// TODO: port validation (StringInSlice ["space","region"])
						},
					},
				},
			},
		},
	}
}

func (r *reviewAppConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *reviewAppConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan reviewAppConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipelineID := plan.PipelineID.ValueString()

	pipeline, err := r.config.Api.PipelineInfo(ctx, pipelineID)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to fetch pipeline %s to enable review apps config", pipelineID),
			err.Error(),
		)
		return
	}

	opts := heroku.ReviewAppConfigEnableOpts{
		Repo: plan.OrgRepo.ValueString(),
	}
	log.Printf("[DEBUG] review app enable - org_repo: %s", opts.Repo)

	automaticReviewApps := plan.AutomaticReviewApps.ValueBool()
	opts.AutomaticReviewApps = &automaticReviewApps
	log.Printf("[DEBUG] review app enable - automatic_review_apps: %v", automaticReviewApps)

	destroyStaleApps := plan.DestroyStaleApps.ValueBool()
	opts.DestroyStaleApps = &destroyStaleApps
	log.Printf("[DEBUG] review app enable - destroy_stale_apps: %v", destroyStaleApps)

	waitForCI := plan.WaitForCI.ValueBool()
	opts.WaitForCi = &waitForCI
	log.Printf("[DEBUG] review app enable - wait_for_ci: %v", waitForCI)

	if !plan.BaseName.IsNull() && !plan.BaseName.IsUnknown() && plan.BaseName.ValueString() != "" {
		baseName := plan.BaseName.ValueString()
		if IsFeatureSupported(pipeline.Generation.Name, "pipeline", "base_name") {
			log.Printf("[DEBUG] review app enable - base_name: %s", baseName)
			opts.BaseName = &baseName
		} else {
			resp.Diagnostics.AddWarning(
				fmt.Sprintf("base_name is not supported for %s generation pipelines, ignoring this setting", pipeline.Generation.Name),
				"",
			)
		}
	}

	if len(plan.DeployTarget) > 0 {
		deployTargetID := plan.DeployTarget[0].ID.ValueString()
		deployTargetType := plan.DeployTarget[0].Type.ValueString()
		log.Printf("[DEBUG] review app enable - deploy_target id: %s", deployTargetID)
		log.Printf("[DEBUG] review app enable - deploy_target type: %s", deployTargetType)
		opts.DeployTarget = (*struct {
			ID   string `json:"id" url:"id,key"`
			Type string `json:"type" url:"type,key"`
		})(&struct {
			ID   string
			Type string
		}{ID: deployTargetID, Type: deployTargetType})
	}

	if !plan.StaleDays.IsNull() && !plan.StaleDays.IsUnknown() {
		staleDays := int(plan.StaleDays.ValueInt64())
		log.Printf("[DEBUG] review app enable - stale_days: %d", staleDays)
		opts.StaleDays = &staleDays
	}

	log.Printf("[DEBUG] Enabling review apps config on pipeline %s", pipelineID)

	_, enableErr := r.config.Api.ReviewAppConfigEnable(ctx, pipelineID, opts)
	if enableErr != nil {
		errorDetail := enableErr.Error()
		if strings.Contains(errorDetail, "Not found") {
			errorDetail = fmt.Sprintf("%s\n\nThis error typically occurs when the pipeline is not connected to a GitHub repository. "+
				"Review apps require the pipeline to be connected to GitHub before configuration can be enabled. "+
				"Please connect your pipeline to a GitHub repository using the Heroku Dashboard or CLI:\n"+
				"  heroku pipelines:connect %s --repo %s", errorDetail, pipeline.Name, opts.Repo)
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to enable review apps config for pipeline %s", pipelineID),
			errorDetail,
		)
		return
	}

	log.Printf("[DEBUG] Enabled review apps config on pipeline %s", pipelineID)

	plan.ID = types.StringValue(pipelineID)

	if err := r.readReviewAppConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading review app config after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *reviewAppConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state reviewAppConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readReviewAppConfig(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading review app config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *reviewAppConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan reviewAppConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state reviewAppConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ID from state during updates.
	plan.ID = state.ID

	opts := heroku.ReviewAppConfigUpdateOpts{}

	if !plan.AutomaticReviewApps.Equal(state.AutomaticReviewApps) {
		automaticReviewApps := plan.AutomaticReviewApps.ValueBool()
		opts.AutomaticReviewApps = &automaticReviewApps
		log.Printf("[DEBUG] review app update - automatic_review_apps: %v", automaticReviewApps)
	}

	if !plan.BaseName.Equal(state.BaseName) {
		baseName := plan.BaseName.ValueString()
		opts.BaseName = &baseName
		log.Printf("[DEBUG] review app update - base_name: %v", baseName)
	}

	if !deployTargetsEqual(plan.DeployTarget, state.DeployTarget) {
		if len(plan.DeployTarget) > 0 {
			deployTargetID := plan.DeployTarget[0].ID.ValueString()
			deployTargetType := plan.DeployTarget[0].Type.ValueString()
			log.Printf("[DEBUG] review app update - deploy_target id: %s", deployTargetID)
			log.Printf("[DEBUG] review app update - deploy_target type: %s", deployTargetType)
			opts.DeployTarget = (*struct {
				ID   string `json:"id" url:"id,key"`
				Type string `json:"type" url:"type,key"`
			})(&struct {
				ID   string
				Type string
			}{ID: deployTargetID, Type: deployTargetType})
		}
	}

	if !plan.DestroyStaleApps.Equal(state.DestroyStaleApps) {
		destroyStaleApps := plan.DestroyStaleApps.ValueBool()
		opts.DestroyStaleApps = &destroyStaleApps
		log.Printf("[DEBUG] review app update - destroy_stale_apps: %v", destroyStaleApps)
	}

	if !plan.StaleDays.Equal(state.StaleDays) {
		staleDays := int(plan.StaleDays.ValueInt64())
		opts.StaleDays = &staleDays
		log.Printf("[DEBUG] review app update - stale_days: %v", staleDays)
	}

	if !plan.WaitForCI.Equal(state.WaitForCI) {
		waitForCI := plan.WaitForCI.ValueBool()
		opts.WaitForCi = &waitForCI
		log.Printf("[DEBUG] review app update - wait_for_ci: %v", waitForCI)
	}

	log.Printf("[DEBUG] Updating review apps config on pipeline %s", state.ID.ValueString())

	_, updateErr := r.config.Api.ReviewAppConfigUpdate(ctx, state.ID.ValueString(), opts)
	if updateErr != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to update review apps config for pipeline %s", state.ID.ValueString()),
			updateErr.Error(),
		)
		return
	}

	log.Printf("[DEBUG] Updated review apps config on pipeline %s", state.ID.ValueString())

	if err := r.readReviewAppConfig(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading review app config after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *reviewAppConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state reviewAppConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[DEBUG] Disabling review apps config on pipeline %s", state.ID.ValueString())

	_, disableErr := r.config.Api.ReviewAppConfigDelete(ctx, state.ID.ValueString())
	if disableErr != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to disable review apps config for pipeline %s", state.ID.ValueString()),
			disableErr.Error(),
		)
		return
	}

	log.Printf("[DEBUG] Disabled review apps config on pipeline %s", state.ID.ValueString())
}

func (r *reviewAppConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pipelineID, orgRepo, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing review app config", "unable to parse import ID for pipeline ID and Github org/repo")
		return
	}

	state := reviewAppConfigResourceModel{
		ID:         types.StringValue(pipelineID),
		PipelineID: types.StringValue(pipelineID),
		OrgRepo:    types.StringValue(orgRepo),
	}

	if err := r.readReviewAppConfig(ctx, &state); err != nil {
		resp.Diagnostics.AddError("Error reading review app config during import", err.Error())
		return
	}

	// Ensure org_repo is preserved from the import ID since the API does not
	// return it directly.
	state.OrgRepo = types.StringValue(orgRepo)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// readReviewAppConfig fetches the current review app configuration for the
// pipeline identified by m.ID and populates the model fields from the API
// response, mirroring the SDKv2 resourceHerokuReviewAppConfigRead logic.
func (r *reviewAppConfigResource) readReviewAppConfig(ctx context.Context, m *reviewAppConfigResourceModel) error {
	reviewAppConfig, err := r.config.Api.ReviewAppConfigInfo(ctx, m.ID.ValueString())
	if err != nil {
		return fmt.Errorf("unable to retrieve review apps config for pipeline %s: %w", m.ID.ValueString(), err)
	}

	m.AutomaticReviewApps = types.BoolValue(reviewAppConfig.AutomaticReviewApps)
	m.DestroyStaleApps = types.BoolValue(reviewAppConfig.DestroyStaleApps)
	m.WaitForCI = types.BoolValue(reviewAppConfig.WaitForCi)
	m.StaleDays = types.Int64Value(int64(reviewAppConfig.StaleDays))
	m.RepoID = types.Int64Value(int64(reviewAppConfig.Repo.ID))

	if reviewAppConfig.BaseName != nil {
		m.BaseName = types.StringValue(*reviewAppConfig.BaseName)
	} else {
		m.BaseName = types.StringValue("")
	}

	if reviewAppConfig.DeployTarget != nil {
		var deployTargetID string
		switch reviewAppConfig.DeployTarget.Type {
		case DeployTargetTypeRegion:
			// The /review-app-config endpoint returns the region UUID for the
			// deploy target ID instead of the name (e.g. "us"). Look up the
			// region to retrieve its human-readable name.
			region, regionErr := r.config.Api.RegionInfo(ctx, reviewAppConfig.DeployTarget.ID)
			if regionErr != nil {
				return fmt.Errorf("unable to retrieve region %s: %w", reviewAppConfig.DeployTarget.ID, regionErr)
			}
			deployTargetID = region.Name
		case DeployTargetTypeSpace:
			deployTargetID = reviewAppConfig.DeployTarget.ID
		default:
			return fmt.Errorf("unknown deploy target type: %s", reviewAppConfig.DeployTarget.Type)
		}

		m.DeployTarget = []reviewAppConfigDeployTargetModel{
			{
				ID:   types.StringValue(deployTargetID),
				Type: types.StringValue(reviewAppConfig.DeployTarget.Type),
			},
		}
	} else {
		m.DeployTarget = []reviewAppConfigDeployTargetModel{}
	}

	return nil
}

// deployTargetsEqual returns true when two deploy target slices contain the
// same single entry (or both are empty), used to detect changes in Update.
func deployTargetsEqual(a, b []reviewAppConfigDeployTargetModel) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return a[0].ID.Equal(b[0].ID) && a[0].Type.Equal(b[0].Type)
}

// validateDeployTargetIDFramework checks that the deploy target ID is either a
// UUID or a 2-letter region code. This mirrors the SDKv2 validateDeployTargetID
// function and is preserved for reference.
//
//nolint:deadcode,unused
func validateDeployTargetIDFramework(value string) bool {
	pattern := `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$|^[a-z]{2}$`
	return regexp.MustCompile(pattern).MatchString(value)
}
