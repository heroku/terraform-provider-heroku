package heroku

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
	tarinator "github.com/verybluebot/tarinator-go"
)

var (
	_ resource.Resource                 = (*buildResource)(nil)
	_ resource.ResourceWithConfigure    = (*buildResource)(nil)
	_ resource.ResourceWithImportState  = (*buildResource)(nil)
	_ resource.ResourceWithUpgradeState = (*buildResource)(nil)
	_ resource.ResourceWithModifyPlan   = (*buildResource)(nil)
)

// NewBuildResource returns the framework implementation of the heroku_build resource.
func NewBuildResource() resource.Resource {
	return &buildResource{}
}

type buildResource struct {
	config *Config
}

// buildSourceModel mirrors the "source" nested block.
type buildSourceModel struct {
	Checksum types.String `tfsdk:"checksum"`
	Path     types.String `tfsdk:"path"`
	URL      types.String `tfsdk:"url"`
	Version  types.String `tfsdk:"version"`
}

// buildUserModel mirrors an element of the "user" computed nested attribute.
type buildUserModel struct {
	Email types.String `tfsdk:"email"`
	ID    types.String `tfsdk:"id"`
}

// buildUserObjectType returns the object type for a "user" list element,
// matching the nested attribute schema.
func buildUserObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"email": types.StringType,
		"id":    types.StringType,
	}}
}

// buildResourceModel is the model for schema version 1.
type buildResourceModel struct {
	ID              types.String       `tfsdk:"id"`
	AppID           types.String       `tfsdk:"app_id"`
	Buildpacks      types.List         `tfsdk:"buildpacks"`
	Source          []buildSourceModel `tfsdk:"source"`
	OutputStreamURL types.String       `tfsdk:"output_stream_url"`
	ReleaseID       types.String       `tfsdk:"release_id"`
	SlugID          types.String       `tfsdk:"slug_id"`
	Stack           types.String       `tfsdk:"stack"`
	Status          types.String       `tfsdk:"status"`
	User            types.List         `tfsdk:"user"`
	UUID            types.String       `tfsdk:"uuid"`
	LocalChecksum   types.String       `tfsdk:"local_checksum"`
}

// buildResourceModelV0 is the model for schema version 0, which used "app" instead of "app_id".
type buildResourceModelV0 struct {
	ID              types.String       `tfsdk:"id"`
	App             types.String       `tfsdk:"app"`
	Buildpacks      types.List         `tfsdk:"buildpacks"`
	Source          []buildSourceModel `tfsdk:"source"`
	OutputStreamURL types.String       `tfsdk:"output_stream_url"`
	ReleaseID       types.String       `tfsdk:"release_id"`
	SlugID          types.String       `tfsdk:"slug_id"`
	Stack           types.String       `tfsdk:"stack"`
	Status          types.String       `tfsdk:"status"`
	User            types.List         `tfsdk:"user"`
	UUID            types.String       `tfsdk:"uuid"`
	LocalChecksum   types.String       `tfsdk:"local_checksum"`
}

func (r *buildResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_build"
}

func (r *buildResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"buildpacks": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"output_stream_url": schema.StringAttribute{
				Computed: true,
			},
			"release_id": schema.StringAttribute{
				Computed: true,
			},
			"slug_id": schema.StringAttribute{
				Computed: true,
			},
			"stack": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"uuid": schema.StringAttribute{
				Computed: true,
			},
			"local_checksum": schema.StringAttribute{
				Computed: true,
			},
			// user is populated by the API after the build is created. It is a
			// Computed nested attribute (not a block) so the framework can plan it
			// as unknown and accept the post-apply value, avoiding the
			// "block count changed from 0 to 1" inconsistency a computed block hits.
			"user": schema.ListNestedAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{
							Computed: true,
						},
						"id": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"source": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"checksum": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"path": schema.StringAttribute{
							Optional: true,
							Validators: []fwvalidator.String{
								// Restore the SDKv2 ConflictsWith: source.path and
								// source.url are mutually exclusive. Without this, a
								// config setting both passes plan and the runtime
								// silently prefers path, ignoring url.
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("url")),
							},
						},
						"url": schema.StringAttribute{
							Optional: true,
							Validators: []fwvalidator.String{
								buildSourceURLValidator{},
							},
						},
						"version": schema.StringAttribute{
							Optional: true,
						},
					},
				},
				// Restore the SDKv2 Required + MaxItems: 1 guarantee: exactly one
				// source block. Create only consumes source[0]; without this,
				// additional blocks would be silently ignored, and zero blocks
				// would only error at apply time.
				Validators: []fwvalidator.List{
					listvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *buildResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan ports resourceHerokuBuildCustomizeDiff:
//  1. Validate buildpacks against the app's generation (skip silently if app unreachable).
//  2. If source.path is set, compute the real checksum; if it differs from the
//     planned local_checksum, update the plan and append local_checksum to RequiresReplace.
func (r *buildResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip on destroy (plan is null).
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan buildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// --- 1. Buildpack generation validation ---
	if !plan.Buildpacks.IsNull() && !plan.Buildpacks.IsUnknown() && len(plan.Buildpacks.Elements()) > 0 {
		appID := plan.AppID.ValueString()
		if appID != "" && r.config != nil {
			app, err := r.config.Api.AppInfo(ctx, appID)
			if err == nil {
				if genErr := frameworkValidateBuildpacksForGenerationAndStack(app.Generation.Name, app.BuildStack.Name); genErr != nil {
					resp.Diagnostics.AddError("Invalid buildpacks for app generation", genErr.Error())
					return
				}
			}
			// If AppInfo fails, skip silently — will be caught at apply time.
		}
	}

	// --- 2. source.path checksum diff ---
	if len(plan.Source) > 0 {
		sourcePath := plan.Source[0].Path.ValueString()
		if sourcePath != "" {
			fileInfo, statErr := os.Stat(sourcePath)
			if statErr != nil {
				resp.Diagnostics.AddError(
					"Error stating build source path",
					fmt.Sprintf("Error stating build source path %s: %s", sourcePath, statErr),
				)
				return
			}

			var realChecksum string
			var csErr error
			if fileInfo.IsDir() {
				realChecksum, csErr = frameworkChecksumSourceRelaxed(sourcePath)
			} else {
				realChecksum, csErr = frameworkChecksumSource(sourcePath)
			}
			if csErr != nil {
				resp.Diagnostics.AddError(
					"Error calculating source checksum",
					csErr.Error(),
				)
				return
			}

			plannedChecksum := plan.LocalChecksum.ValueString()
			log.Printf("[DEBUG] build ModifyPlan: planned local_checksum '%s', real '%s'", plannedChecksum, realChecksum)

			if plannedChecksum != realChecksum {
				plan.LocalChecksum = types.StringValue(realChecksum)
				resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("local_checksum"))
			}
		}
	}
}

func (r *buildResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan buildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()
	client := r.config.Api

	// Apply-time validation: ensure buildpacks are not specified for Fir apps.
	if !plan.Buildpacks.IsNull() && !plan.Buildpacks.IsUnknown() && len(plan.Buildpacks.Elements()) > 0 {
		app, err := client.AppInfo(ctx, appID)
		if err != nil {
			resp.Diagnostics.AddError("Error fetching app info for buildpack validation", err.Error())
			return
		}
		if err := frameworkValidateBuildpacksForGenerationAndStack(app.Generation.Name, app.BuildStack.Name); err != nil {
			resp.Diagnostics.AddError("Invalid buildpacks for app generation", err.Error())
			return
		}
	}

	opts := heroku.BuildCreateOpts{}

	// Buildpacks
	if !plan.Buildpacks.IsNull() && !plan.Buildpacks.IsUnknown() && len(plan.Buildpacks.Elements()) > 0 {
		var bpStrings []string
		resp.Diagnostics.Append(plan.Buildpacks.ElementsAs(ctx, &bpStrings, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var buildpacks []*struct {
			Name *string `json:"name,omitempty" url:"name,omitempty,key"`
			URL  *string `json:"url,omitempty" url:"url,omitempty,key"`
		}
		for _, b := range bpStrings {
			bCopy := b
			buildpacks = append(buildpacks, &struct {
				Name *string `json:"name,omitempty" url:"name,omitempty,key"`
				URL  *string `json:"url,omitempty" url:"url,omitempty,key"`
			}{
				URL: &bCopy,
			})
		}
		opts.Buildpacks = buildpacks
	}

	// Source
	var localChecksum string
	if len(plan.Source) == 0 {
		resp.Diagnostics.AddError(
			"Invalid source configuration",
			"Build requires a source block with either path or url",
		)
		return
	}

	src := plan.Source[0]

	if !src.Checksum.IsNull() && !src.Checksum.IsUnknown() && src.Checksum.ValueString() != "" {
		if !src.Path.IsNull() && src.Path.ValueString() != "" {
			resp.Diagnostics.AddError(
				"Invalid source configuration",
				"source.checksum should be empty when source.path is set (checksum is auto-generated)",
			)
			return
		}
		s := src.Checksum.ValueString()
		opts.SourceBlob.Checksum = &s
	}

	if !src.Version.IsNull() && !src.Version.IsUnknown() && src.Version.ValueString() != "" {
		s := src.Version.ValueString()
		opts.SourceBlob.Version = &s
	}

	if !src.Path.IsNull() && src.Path.ValueString() != "" {
		sourcePath := src.Path.ValueString()
		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error stating build source path",
				fmt.Sprintf("Error stating build source path %s: %s", sourcePath, err),
			)
			return
		}
		useRelaxedChecksum := fileInfo.IsDir()

		var tarballPath string
		if useRelaxedChecksum {
			tarballPath, err = frameworkGenerateSourceTarball(sourcePath)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error generating build source tarball",
					fmt.Sprintf("Error generating build source tarball %s: %s", sourcePath, err),
				)
				return
			}
			defer frameworkCleanupSourceFile(tarballPath)
			localChecksum, err = frameworkChecksumSourceRelaxed(sourcePath)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error calculating relaxed checksum",
					fmt.Sprintf("Error calculating relaxed checksum for directory source %s: %s", sourcePath, err),
				)
				return
			}
		} else {
			tarballPath = sourcePath
			localChecksum, err = frameworkChecksumSource(tarballPath)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error calculating checksum",
					fmt.Sprintf("Error calculating checksum for tarball source %s: %s", tarballPath, err),
				)
				return
			}
		}

		newSource, err := client.SourceCreate(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error creating source for build", err.Error())
			return
		}
		err = frameworkUploadSource(tarballPath, "PUT", newSource.SourceBlob.PutURL)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error uploading source for build",
				fmt.Sprintf("Error uploading source for build to %s: %s", newSource.SourceBlob.PutURL, err),
			)
			return
		}
		opts.SourceBlob.URL = &newSource.SourceBlob.GetURL
		if !useRelaxedChecksum {
			opts.SourceBlob.Checksum = &localChecksum
		}
	} else if !src.URL.IsNull() && src.URL.ValueString() != "" {
		s := src.URL.ValueString()
		opts.SourceBlob.URL = &s
	} else {
		resp.Diagnostics.AddError(
			"Invalid source configuration",
			"Build requires either source.path or source.url",
		)
		return
	}

	build, err := client.BuildCreate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating build",
			fmt.Sprintf("Error creating build: %s opts %+v", err, opts),
		)
		return
	}

	// Wait for the build to complete — up to 26 hours, honouring ctx cancellation.
	log.Printf("[DEBUG] Waiting for Build (%s:%s) to complete", appID, build.ID)
	if _, waitErr := frameworkWaitForBuild(ctx, client, appID, build.ID, 26*time.Hour); waitErr != nil {
		resp.Diagnostics.AddError("Build did not complete successfully", waitErr.Error())
		return
	}

	// Final read after completion.
	finalBuild, err := client.BuildInfo(ctx, appID, build.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error refreshing completed build", err.Error())
		return
	}

	plan.ID = types.StringValue(build.ID)
	plan.LocalChecksum = types.StringValue(localChecksum)
	populateBuildModel(ctx, &plan, finalBuild, appID, localChecksum)

	log.Printf("[INFO] Created build ID: %s", plan.ID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *buildResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state buildResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := state.AppID.ValueString()
	build, err := r.config.Api.BuildInfo(ctx, appID, state.ID.ValueString())
	if err != nil {
		if isBuildNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading build", err.Error())
		return
	}

	// Preserve local_checksum from state (not returned by the API).
	localChecksum := state.LocalChecksum.ValueString()
	populateBuildModel(ctx, &state, build, appID, localChecksum)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete is a no-op: there is no DELETE build endpoint in the Heroku Platform API.
// Update is required by the resource.Resource interface. Every configurable
// attribute on heroku_build forces replacement, so a plan never produces an
// in-place update in practice; this simply persists the planned values.
func (r *buildResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan buildResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *buildResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	log.Printf("[INFO] There is no DELETE for build resource so this is a no-op. Build will be removed from state.")
}

func (r *buildResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, buildID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing build", err.Error())
		return
	}

	build, err := r.config.Api.BuildInfo(ctx, app, buildID)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching build during import", err.Error())
		return
	}

	var state buildResourceModel
	state.ID = types.StringValue(build.ID)
	// local_checksum is not known from the API on import; leave it empty.
	populateBuildModel(ctx, &state, build, build.App.ID, "")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// UpgradeState handles the v0→v1 migration that renames "app" to "app_id",
// resolving the app name to a UUID if needed. Mirrors upgradeAppToAppID.
func (r *buildResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuBuildV0(): uses "app" instead of "app_id".
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true},
			"app":               schema.StringAttribute{Required: true},
			"output_stream_url": schema.StringAttribute{Computed: true},
			"release_id":        schema.StringAttribute{Computed: true},
			"slug_id":           schema.StringAttribute{Computed: true},
			"stack":             schema.StringAttribute{Computed: true},
			"status":            schema.StringAttribute{Computed: true},
			"uuid":              schema.StringAttribute{Computed: true},
			"local_checksum":    schema.StringAttribute{Computed: true},
			"buildpacks": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"source": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"checksum": schema.StringAttribute{Optional: true, Computed: true},
						"path":     schema.StringAttribute{Optional: true},
						"url":      schema.StringAttribute{Optional: true},
						"version":  schema.StringAttribute{Optional: true},
					},
				},
			},
			"user": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{Computed: true},
						"id":    schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior buildResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading build state", err.Error())
					return
				}

				upgraded := buildResourceModel{
					ID:              prior.ID,
					AppID:           types.StringValue(appID),
					Buildpacks:      prior.Buildpacks,
					Source:          prior.Source,
					OutputStreamURL: prior.OutputStreamURL,
					ReleaseID:       prior.ReleaseID,
					SlugID:          prior.SlugID,
					Stack:           prior.Stack,
					Status:          prior.Status,
					User:            prior.User,
					UUID:            prior.UUID,
					LocalChecksum:   prior.LocalChecksum,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// --- Model population helper ---

// populateBuildModel maps a *heroku.Build API response into the framework model.
// localChecksum is preserved from prior state (or from Create) since the API does not return it.
func populateBuildModel(ctx context.Context, m *buildResourceModel, build *heroku.Build, appID, localChecksum string) {
	m.AppID = types.StringValue(appID)
	m.UUID = types.StringValue(build.ID)
	m.OutputStreamURL = types.StringValue(build.OutputStreamURL)
	m.Stack = types.StringValue(build.Stack)
	m.Status = types.StringValue(build.Status)
	m.LocalChecksum = types.StringValue(localChecksum)

	if build.Release != nil {
		m.ReleaseID = types.StringValue(build.Release.ID)
	} else {
		m.ReleaseID = types.StringValue("")
	}

	if build.Slug != nil {
		m.SlugID = types.StringValue(build.Slug.ID)
	} else {
		m.SlugID = types.StringValue("")
	}

	// Buildpacks
	var bpElems []attr.Value
	for _, bp := range build.Buildpacks {
		bpElems = append(bpElems, types.StringValue(bp.URL))
	}
	if len(bpElems) > 0 {
		listVal, diags := types.ListValueFrom(ctx, types.StringType, bpElems)
		if !diags.HasError() {
			m.Buildpacks = listVal
		}
	} else {
		m.Buildpacks = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// User: a Computed nested attribute, built as a types.List so the framework
	// can plan it as unknown before apply.
	userList, userDiags := types.ListValueFrom(ctx, buildUserObjectType(), []buildUserModel{
		{
			Email: types.StringValue(build.User.Email),
			ID:    types.StringValue(build.User.ID),
		},
	})
	if !userDiags.HasError() {
		m.User = userList
	}

	// Source: preserve path from existing model; only populate checksum/url from
	// the API when path is not set (mirrors SDKv2 setBuildState logic).
	if len(m.Source) > 0 {
		src := m.Source[0]
		// checksum is Optional+Computed, so it plans as unknown when absent from
		// config. It must be set to a known value (or null) after apply.
		if src.Path.IsNull() || src.Path.ValueString() == "" {
			// URL-based source: take the checksum from the API, or null if the
			// API didn't return one.
			if build.SourceBlob.Checksum != nil {
				src.Checksum = types.StringValue(*build.SourceBlob.Checksum)
			} else {
				src.Checksum = types.StringNull()
			}
			if build.SourceBlob.URL != "" {
				src.URL = types.StringValue(build.SourceBlob.URL)
			}
		} else {
			// Path-based source: checksum is auto-generated for the upload and is
			// not a meaningful persisted value here (it lives in local_checksum).
			src.Checksum = types.StringNull()
		}
		if build.SourceBlob.Version != nil {
			src.Version = types.StringValue(*build.SourceBlob.Version)
		}
		m.Source[0] = src
	}
}

// --- Build polling ---

// frameworkWaitForBuild polls BuildInfo until the build leaves "pending" state,
// returning an error if the build fails or the timeout/context is exceeded.
// This mirrors the SDKv2 BuildStateRefreshFunc + StateChangeConf logic.
func frameworkWaitForBuild(ctx context.Context, client *heroku.Service, appID, buildID string, timeout time.Duration) (*heroku.Build, error) {
	deadline := time.Now().Add(timeout)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for build %s (app: %s)", buildID, appID)
		default:
		}

		build, err := client.BuildInfo(ctx, appID, buildID)
		if err != nil {
			return nil, fmt.Errorf("error polling build %s (app: %s): %w", buildID, appID, err)
		}

		switch build.Status {
		case "succeeded":
			return build, nil
		case "failed":
			buildLog := frameworkFetchBuildLog(build.OutputStreamURL)
			return nil, fmt.Errorf("Build failed (app: %s), complete build log follows:\n%s\n(End of build log)", appID, buildLog)
		case "pending":
			log.Printf("[DEBUG] Build pending (app: %s, build: %s)", appID, buildID)
		default:
			log.Printf("[DEBUG] Build status %q (app: %s, build: %s)", build.Status, appID, buildID)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for build %s (app: %s) after %s", buildID, appID, timeout)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for build %s (app: %s)", buildID, appID)
		case <-time.After(10 * time.Second):
		}
	}
}

// frameworkFetchBuildLog fetches the build output stream and returns it as a string.
func frameworkFetchBuildLog(outputStreamURL string) string {
	resp, err := http.Get(outputStreamURL) //nolint:noctx
	if err != nil {
		return fmt.Sprintf("(failed to fetch build logs: %s)", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// --- Ported file-local helpers ---
// These are ported from resource_heroku_build.go, which will be deleted after
// this file is in place. They are prefixed with "framework" to avoid any
// transitional symbol conflicts.

// frameworkValidateBuildpacksForGenerationAndStack validates that traditional
// buildpacks are not used with CNB apps. Fir apps always use CNB; Cedar apps
// use CNB when stack = "cnb". Mirrors validateBuildpacksForGenerationAndStack.
func frameworkValidateBuildpacksForGenerationAndStack(generationName, stackName string) error {
	gen := generationName
	if gen == "" {
		gen = "cedar"
	}
	if !IsFeatureSupported(gen, "app", "buildpacks") || IsCNBApp(gen, stackName) {
		return fmt.Errorf(
			"buildpacks cannot be specified for apps using Cloud Native Buildpacks. " +
				"Use project.toml instead. " +
				"See: https://devcenter.heroku.com/articles/using-multiple-buildpacks-for-an-app",
		)
	}
	return nil
}

// frameworkChecksumSource computes a SHA256 checksum of a single file.
// Mirrors checksumSource.
func frameworkChecksumSource(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("Error opening source.path: %s", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("Error reading source for checksum: %s", err)
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return fmt.Sprintf("SHA256:%x", hash.Sum(nil)), nil
}

// frameworkChecksumSourceRelaxed computes a checksum over filenames and contents
// of a directory tree. Mirrors checksumSourceRelaxed.
func frameworkChecksumSourceRelaxed(sourcePath string) (string, error) {
	hash := sha256.New()

	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("Error stating source path '%s' for checksum %w", sourcePath, err)
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(sourcePath)
	}

	err = filepath.Walk(sourcePath, func(p string, fi os.FileInfo, walkErr error) error {
		var walkPath string
		if baseDir != "" {
			walkPath = filepath.ToSlash(filepath.Join(baseDir, strings.TrimPrefix(p, sourcePath)))
		} else {
			walkPath = fi.Name()
		}

		if walkErr != nil {
			return fmt.Errorf("Error walking '%s' for checksum: %w", walkPath, walkErr)
		}

		fmt.Fprint(hash, walkPath+"\n")
		log.Printf("[DEBUG] hash ← %s (name)", walkPath)

		if fi.IsDir() || !fi.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("Error opening file '%s' for checksum: %w", fi.Name(), err)
		}
		defer f.Close()
		b, err := io.Copy(hash, f)
		if err != nil {
			return fmt.Errorf("Error reading file '%s' for checksum: %w", fi.Name(), err)
		}
		log.Printf("[DEBUG] hash ← %v (bytes)", b)
		return nil
	})
	if err != nil {
		return "", err
	}

	checksum := fmt.Sprintf("SHA256:%x", hash.Sum(nil))
	log.Printf("[DEBUG] hash sum → %v", checksum)
	return checksum, nil
}

// frameworkUploadSource uploads a file to a pre-signed URL.
// Mirrors uploadSource.
func frameworkUploadSource(filePath, httpMethod, httpURL string) error {
	method := strings.ToUpper(httpMethod)
	log.Printf("[DEBUG] Uploading source '%s' to %s %s", filePath, method, httpURL)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening source.path: %s", err)
	}
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Error stating source.path: %s", err)
	}
	defer file.Close()

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, httpURL, file)
	if err != nil {
		return fmt.Errorf("Error creating source upload request: %s", err)
	}
	req.ContentLength = stat.Size()
	log.Printf("[DEBUG] Upload source request: %+v", req)

	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error uploading source: %s", err)
	}

	if b, dumpErr := httputil.DumpResponse(res, true); dumpErr == nil {
		log.Printf("[DEBUG] Source upload response: %s", b)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Unsuccessful HTTP response from source upload: %s", res.Status)
	}
	return nil
}

// frameworkGenerateSourceTarball creates a temp tarball from a directory.
// Mirrors generateSourceTarball.
func frameworkGenerateSourceTarball(dirPath string) (string, error) {
	fi, err := ioutil.TempFile("", "terraform-heroku_build-source-*.tar.gz")
	if err != nil {
		return "", err
	}
	if err := fi.Close(); err != nil {
		return "", err
	}
	tf := fi.Name()
	if err = tarinator.Tarinate([]string{dirPath}, tf); err != nil {
		err = fmt.Errorf("Error generating build source tarball %s of %s: %s", tf, dirPath, err)
	}
	return tf, err
}

// frameworkCleanupSourceFile removes a temp file if it exists.
// Mirrors cleanupSourceFile.
func frameworkCleanupSourceFile(filePath string) {
	if filePath != "" {
		if err := os.Remove(filePath); err != nil {
			log.Printf("[WARN] Error cleaning-up build source tarball: %s (%s)", err, filePath)
		}
	}
}

// buildSourceURLValidator validates at plan time that source.url is a secure
// (https) URL, mirroring the SDKv2 validateSourceUrl ValidateFunc. Without it,
// an insecure URL falls through to the Heroku API and errors only at apply time.
type buildSourceURLValidator struct{}

func (v buildSourceURLValidator) Description(_ context.Context) string {
	return "source.url must be a secure URL starting with https://"
}

func (v buildSourceURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v buildSourceURLValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if err := frameworkValidateSourceURL(value); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid source.url", err.Error())
	}
}

// frameworkValidateSourceURL checks that a source URL starts with https://.
// Mirrors validateSourceUrl.
func frameworkValidateSourceURL(value string) error {
	pattern := `^https://`
	if !regexp.MustCompile(pattern).MatchString(value) {
		return fmt.Errorf("source.url must be a secure URL, starting with `https://`. Value is %q", value)
	}
	return nil
}

// isBuildNotFound returns true when a heroku-go error represents a 404 response.
func isBuildNotFound(err error) bool {
	if err == nil {
		return false
	}
	herokuErr, ok := err.(heroku.Error)
	return ok && herokuErr.StatusCode == 404
}
