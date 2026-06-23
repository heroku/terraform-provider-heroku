package heroku

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                 = (*slugResource)(nil)
	_ resource.ResourceWithConfigure    = (*slugResource)(nil)
	_ resource.ResourceWithImportState  = (*slugResource)(nil)
	_ resource.ResourceWithUpgradeState = (*slugResource)(nil)
	_ resource.ResourceWithModifyPlan   = (*slugResource)(nil)
)

// NewSlugResource returns the framework implementation of the heroku_slug resource.
func NewSlugResource() resource.Resource {
	return &slugResource{}
}

type slugResource struct {
	config *Config
}

// slugBlobAttrTypes is the attribute type map for the blob nested object, used when
// constructing types.Object values in slugPopulateModel.
var slugBlobAttrTypes = map[string]attr.Type{
	"method": types.StringType,
	"url":    types.StringType,
}

// slugResourceModel is the Terraform state model for the current schema (version 1).
type slugResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	AppID                        types.String `tfsdk:"app_id"`
	FilePath                     types.String `tfsdk:"file_path"`
	FileURL                      types.String `tfsdk:"file_url"`
	Blob                         types.Object `tfsdk:"blob"`
	BuildpackProvidedDescription types.String `tfsdk:"buildpack_provided_description"`
	Checksum                     types.String `tfsdk:"checksum"`
	Commit                       types.String `tfsdk:"commit"`
	CommitDescription            types.String `tfsdk:"commit_description"`
	ProcessTypes                 types.Map    `tfsdk:"process_types"`
	Size                         types.Int64  `tfsdk:"size"`
	Stack                        types.String `tfsdk:"stack"`
	StackID                      types.String `tfsdk:"stack_id"`
}

// slugResourceModelV0 is the Terraform state model for the prior schema (version 0),
// which used "app" instead of "app_id".
type slugResourceModelV0 struct {
	ID                           types.String `tfsdk:"id"`
	App                          types.String `tfsdk:"app"`
	FilePath                     types.String `tfsdk:"file_path"`
	FileURL                      types.String `tfsdk:"file_url"`
	Blob                         types.Object `tfsdk:"blob"`
	BuildpackProvidedDescription types.String `tfsdk:"buildpack_provided_description"`
	Checksum                     types.String `tfsdk:"checksum"`
	Commit                       types.String `tfsdk:"commit"`
	CommitDescription            types.String `tfsdk:"commit_description"`
	ProcessTypes                 types.Map    `tfsdk:"process_types"`
	Size                         types.Int64  `tfsdk:"size"`
	Stack                        types.String `tfsdk:"stack"`
	StackID                      types.String `tfsdk:"stack_id"`
}

// slugFileURLValidator checks that file_url starts with "https://", mirroring
// the SDK validateFileUrl function.
type slugFileURLValidator struct{}

func (v slugFileURLValidator) Description(_ context.Context) string {
	return "file_url must be a secure URL starting with https://"
}

func (v slugFileURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v slugFileURLValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	pattern := `^https://`
	if !regexp.MustCompile(pattern).MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid file_url",
			fmt.Sprintf("%q must be a secure URL starting with `https://`. Value is %q", req.Path, value),
		)
	}
}

// slugProcessTypesRequiresReplace is a planmodifier.Map that marks the resource for
// replacement whenever process_types changes, mirroring ForceNew in the SDKv2 schema.
type slugProcessTypesRequiresReplace struct{}

func (m slugProcessTypesRequiresReplace) Description(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m slugProcessTypesRequiresReplace) MarkdownDescription(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m slugProcessTypesRequiresReplace) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.RequiresReplace = true
}

// slugConflictsWithValidator is an inline ConflictsWith validator for string
// attributes, mirroring the conflictsWithStringValidator used in framework_resource_drain.go.
type slugConflictsWithValidator struct {
	expressions path.Expressions
}

func (v slugConflictsWithValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Conflicts with attributes: %v", v.expressions)
}

func (v slugConflictsWithValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v slugConflictsWithValidator) ValidateString(ctx context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, expr := range v.expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expr)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			var otherVal types.String
			diags = req.Config.GetAttribute(ctx, mp, &otherVal)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				continue
			}

			if otherVal.IsNull() || otherVal.IsUnknown() {
				continue
			}

			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Conflicting Attribute",
				fmt.Sprintf("Attribute %q conflicts with attribute %q. Only one may be set.", req.Path, mp),
			)
		}
	}
}

func (r *slugResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slug"
}

func (r *slugResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"file_path": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					slugConflictsWithValidator{
						expressions: path.Expressions{path.MatchRoot("file_url")},
					},
				},
			},
			"file_url": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					slugFileURLValidator{},
					slugConflictsWithValidator{
						expressions: path.Expressions{path.MatchRoot("file_path")},
					},
				},
			},
			"blob": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"method": schema.StringAttribute{
						Computed: true,
					},
					"url": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"buildpack_provided_description": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"checksum": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"commit": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"commit_description": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"process_types": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					slugProcessTypesRequiresReplace{},
				},
			},
			"size": schema.Int64Attribute{
				Computed: true,
			},
			"stack": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"stack_id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *slugResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan ports resourceHerokuSlugCustomizeDiff: when file_path is set,
// compute the real SHA256 checksum of the file; if it differs from the planned
// checksum, update the plan value and force replacement.
func (r *slugResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip on destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan slugResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FilePath.IsNull() || plan.FilePath.IsUnknown() || plan.FilePath.ValueString() == "" {
		return
	}

	filePath := plan.FilePath.ValueString()
	realChecksum, err := slugChecksumFile(filePath)
	if err != nil {
		// Non-fatal: the file might not exist yet (e.g. during plan without apply).
		log.Printf("[DEBUG] slugResource ModifyPlan: could not checksum %s: %s", filePath, err)
		return
	}

	var stateChecksum string
	if !req.State.Raw.IsNull() {
		var state slugResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateChecksum = state.Checksum.ValueString()
	}

	newChecksum := plan.Checksum.ValueString()
	log.Printf("[DEBUG] Diffing slug: old '%s', new '%s', real '%s'", stateChecksum, newChecksum, realChecksum)

	if newChecksum != realChecksum {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("checksum"), realChecksum)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("checksum"))
	}
}

func (r *slugResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan slugResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()

	// Build creation options.
	opts := heroku.SlugCreateOpts{}

	// process_types is required; convert from types.Map.
	var processTypesMap map[string]string
	resp.Diagnostics.Append(plan.ProcessTypes.ElementsAs(ctx, &processTypesMap, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	opts.ProcessTypes = processTypesMap

	if !plan.BuildpackProvidedDescription.IsNull() && !plan.BuildpackProvidedDescription.IsUnknown() {
		v := plan.BuildpackProvidedDescription.ValueString()
		opts.BuildpackProvidedDescription = heroku.String(v)
	}

	// Only file_path or file_url will be set, because of ConflictsWith validators.
	var filePath string

	if !plan.FilePath.IsNull() && !plan.FilePath.IsUnknown() && plan.FilePath.ValueString() != "" {
		filePath = plan.FilePath.ValueString()
	}

	if !plan.FileURL.IsNull() && !plan.FileURL.IsUnknown() && plan.FileURL.ValueString() != "" {
		fileURL := plan.FileURL.ValueString()

		newUUID, err := uuid.GenerateUUID()
		if err != nil {
			resp.Diagnostics.AddError("Error generating UUID for slug file", err.Error())
			return
		}
		filePath = fmt.Sprintf("slug-%s.tgz", newUUID)

		if err := slugDownloadFile(fileURL, filePath); err != nil {
			resp.Diagnostics.AddError("Error downloading slug", err.Error())
			return
		}
		defer slugCleanupFile(filePath)
	}

	if filePath == "" {
		resp.Diagnostics.AddError(
			"Error creating slug",
			"requires either `file_path` or `file_url` attribute",
		)
		return
	}

	// Determine checksum: use the user-specified value if set, otherwise compute it.
	if !plan.Checksum.IsNull() && !plan.Checksum.IsUnknown() && plan.Checksum.ValueString() != "" {
		opts.Checksum = heroku.String(plan.Checksum.ValueString())
	} else {
		checksum, err := slugChecksumFile(filePath)
		if err != nil {
			resp.Diagnostics.AddError("Error computing slug checksum", err.Error())
			return
		}
		log.Printf("[DEBUG] Slug checksum: %s", checksum)
		opts.Checksum = heroku.String(checksum)
	}

	if !plan.Commit.IsNull() && !plan.Commit.IsUnknown() {
		opts.Commit = heroku.String(plan.Commit.ValueString())
	}
	if !plan.CommitDescription.IsNull() && !plan.CommitDescription.IsUnknown() {
		opts.CommitDescription = heroku.String(plan.CommitDescription.ValueString())
	}
	if !plan.Stack.IsNull() && !plan.Stack.IsUnknown() && plan.Stack.ValueString() != "" {
		opts.Stack = heroku.String(plan.Stack.ValueString())
	}

	slug, err := r.config.Api.SlugCreate(ctx, appID, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating slug",
			fmt.Sprintf("%s opts %+v", err, opts),
		)
		return
	}

	// Upload the slug archive. Retry until context timeout or success, sleeping
	// 10s between attempts (mirrors the SDK resource.Retry / 10s-sleep loop).
	if filePath != "" {
		var uploadErr error
		deadline := time.Now().Add(20 * time.Minute)
		for {
			uploadErr = slugUploadFile(filePath, slug.Blob.Method, slug.Blob.URL)
			if uploadErr == nil {
				break
			}
			log.Printf("[DEBUG] Error uploading slug: %s", uploadErr.Error())
			if time.Now().After(deadline) {
				resp.Diagnostics.AddError(
					"Error uploading slug",
					fmt.Sprintf("upload timed out; last error: %s", uploadErr),
				)
				return
			}
			log.Printf("[DEBUG] Retrying slug upload in 10s")
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError(
					"Error uploading slug",
					fmt.Sprintf("context cancelled during upload; last error: %s", uploadErr),
				)
				return
			case <-time.After(10 * time.Second):
			}
		}
	}

	plan.ID = types.StringValue(slug.ID)

	// Re-read the slug after upload so the persisted state matches what a
	// subsequent Read/import would return. The SlugCreate response is a
	// pre-upload snapshot whose server-computed fields (notably "size") differ
	// from SlugInfo's authoritative values; populating from it directly caused
	// ImportStateVerify drift (create "size"=324 vs import "size"=0).
	createdSlug, err := r.config.Api.SlugInfo(ctx, appID, slug.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading slug after create",
			fmt.Sprintf("Error retrieving slug %s: %s", slug.ID, err),
		)
		return
	}
	slugPopulateModel(ctx, &plan, createdSlug)

	log.Printf("[INFO] Created slug ID: %s", plan.ID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *slugResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state slugResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug, err := r.config.Api.SlugInfo(ctx, state.AppID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "not_found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error retrieving slug",
			fmt.Sprintf("Error retrieving slug %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	slugPopulateModel(ctx, &state, slug)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op: all writable fields are ForceNew (RequiresReplace), so
// Terraform will destroy-and-recreate rather than call Update. This method
// exists to satisfy the resource.Resource interface.
func (r *slugResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state slugResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete is a no-op because the Heroku Platform API has no DELETE slug endpoint.
// The resource is simply removed from state, mirroring resourceHerokuSlugDelete.
func (r *slugResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	log.Printf("[INFO] There is no DELETE for slug resource so this is a no-op. Slug will be removed from state.")
}

// ImportState handles "terraform import heroku_slug.x app:slugID".
// It resolves the app name/ID to the app UUID via resourceHerokuAppRetrieve,
// mirroring resourceHerokuSlugImport.
func (r *slugResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, slugID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing slug", err.Error())
		return
	}

	slug, err := r.config.Api.SlugInfo(ctx, app, slugID)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching slug during import", err.Error())
		return
	}

	foundApp, err := resourceHerokuAppRetrieve(app, r.config.Api)
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving app during slug import", err.Error())
		return
	}

	state := slugResourceModel{
		ID:    types.StringValue(slug.ID),
		AppID: types.StringValue(foundApp.App.ID),
		// file_path and file_url are not recoverable on import; leave null.
		FilePath: types.StringNull(),
		FileURL:  types.StringNull(),
	}
	slugPopulateModel(ctx, &state, slug)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// UpgradeState handles the v0→v1 migration that renames "app" to "app_id",
// resolving the app name to a UUID via resolveAppToAppID. This mirrors the
// SDKv2 upgradeAppToAppID StateUpgrader registered against resourceHerokuSlugV0.
func (r *slugResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema mirrors resourceHerokuSlugV0(): same as v1 but with "app" instead of "app_id".
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"file_path": schema.StringAttribute{
				Optional: true,
			},
			"file_url": schema.StringAttribute{
				Optional: true,
			},
			"blob": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"method": schema.StringAttribute{Computed: true},
					"url":    schema.StringAttribute{Computed: true},
				},
			},
			"buildpack_provided_description": schema.StringAttribute{
				Optional: true,
			},
			"checksum": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"commit": schema.StringAttribute{
				Optional: true,
			},
			"commit_description": schema.StringAttribute{
				Optional: true,
			},
			"process_types": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
			"size": schema.Int64Attribute{
				Computed: true,
			},
			"stack": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"stack_id": schema.StringAttribute{
				Computed: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior slugResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error upgrading slug state", err.Error())
					return
				}

				upgraded := slugResourceModel{
					ID:                           prior.ID,
					AppID:                        types.StringValue(appID),
					FilePath:                     prior.FilePath,
					FileURL:                      prior.FileURL,
					Blob:                         prior.Blob,
					BuildpackProvidedDescription: prior.BuildpackProvidedDescription,
					Checksum:                     prior.Checksum,
					Commit:                       prior.Commit,
					CommitDescription:            prior.CommitDescription,
					ProcessTypes:                 prior.ProcessTypes,
					Size:                         prior.Size,
					Stack:                        prior.Stack,
					StackID:                      prior.StackID,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// slugPopulateModel populates the slug resource model from an API slug response.
// It mirrors setSlugState from resource_heroku_slug.go.
func slugPopulateModel(_ context.Context, m *slugResourceModel, slug *heroku.Slug) {
	blobObj, _ := types.ObjectValue(slugBlobAttrTypes, map[string]attr.Value{
		"method": types.StringValue(slug.Blob.Method),
		"url":    types.StringValue(slug.Blob.URL),
	})
	m.Blob = blobObj

	if slug.BuildpackProvidedDescription != nil {
		m.BuildpackProvidedDescription = types.StringValue(*slug.BuildpackProvidedDescription)
	} else {
		m.BuildpackProvidedDescription = types.StringNull()
	}

	if slug.Checksum != nil {
		m.Checksum = types.StringValue(*slug.Checksum)
	} else {
		m.Checksum = types.StringNull()
	}

	if slug.Commit != nil {
		m.Commit = types.StringValue(*slug.Commit)
	} else {
		m.Commit = types.StringNull()
	}

	if slug.CommitDescription != nil {
		m.CommitDescription = types.StringValue(*slug.CommitDescription)
	} else {
		m.CommitDescription = types.StringNull()
	}

	if slug.Size != nil {
		m.Size = types.Int64Value(int64(*slug.Size))
	} else {
		m.Size = types.Int64Value(0)
	}

	m.StackID = types.StringValue(slug.Stack.ID)
	m.Stack = types.StringValue(slug.Stack.Name)

	// process_types is a map[string]string on the API response.
	ptVals := make(map[string]attr.Value, len(slug.ProcessTypes))
	for k, v := range slug.ProcessTypes {
		ptVals[k] = types.StringValue(v)
	}
	ptMap, _ := types.MapValue(types.StringType, ptVals)
	m.ProcessTypes = ptMap
}

// ---------------------------------------------------------------------------
// File helpers — ported from resource_heroku_slug.go with slug-prefixed names
// to avoid symbol collisions when the original file is deleted.
// ---------------------------------------------------------------------------

// slugDownloadFile downloads the tarball at httpUrl to destinationFilePath.
// Mirrors downloadSlug.
func slugDownloadFile(httpURL, destinationFilePath string) error {
	log.Printf("[DEBUG] Downloading slug from %s", httpURL)

	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		return fmt.Errorf("Error creating slug download request: %s (%s)", err, httpURL)
	}
	log.Printf("[DEBUG] Download slug request: %+v", req)
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error downloading slug: %s (%s)", err, httpURL)
	}

	b, err := httputil.DumpResponse(res, true)
	if err == nil {
		log.Printf("[DEBUG] Slug download response: %s", b)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Unsuccessful HTTP response from slug download: %s (%s)", res.Status, httpURL)
	}

	slugFile, err := os.Create(destinationFilePath)
	if err != nil {
		return fmt.Errorf("Error creating slug file: %s (%s)", err, destinationFilePath)
	}
	defer slugFile.Close()

	if _, copyErr := io.Copy(slugFile, res.Body); copyErr != nil {
		return copyErr
	}

	return nil
}

// slugUploadFile uploads the file at filePath to the given HTTP endpoint.
// Mirrors uploadSlug.
func slugUploadFile(filePath, httpMethod, httpURL string) error {
	method := strings.ToUpper(httpMethod)
	log.Printf("[DEBUG] Uploading slug '%s' to %s %s", filePath, method, httpURL)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error opening slug file_path: %s", err)
	}
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Error stating slug file_path: %s", err)
	}
	defer file.Close()

	httpClient := &http.Client{}
	req, err := http.NewRequest(method, httpURL, file)
	if err != nil {
		return fmt.Errorf("Error creating slug upload request: %s", err)
	}
	req.ContentLength = stat.Size()
	log.Printf("[DEBUG] Upload slug request: %+v", req)
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error uploading slug: %s", err)
	}

	b, err := httputil.DumpResponse(res, true)
	if err == nil {
		log.Printf("[DEBUG] Slug upload response: %s", b)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Unsuccessful HTTP response from slug upload: %s", res.Status)
	}
	return nil
}

// slugChecksumFile computes the SHA256 checksum of the file at filePath,
// returning "SHA256:<hex>". Mirrors checksumSlug.
func slugChecksumFile(filePath string) (string, error) {
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return "", fmt.Errorf("Error opening slug file_path: %s", openErr)
	}

	hash := sha256.New()
	if _, copyErr := io.Copy(hash, file); copyErr != nil {
		return "", fmt.Errorf("Error reading slug for checksum: %s", copyErr)
	}

	closeErr := file.Close()
	if closeErr != nil {
		return "", closeErr
	}

	return fmt.Sprintf("SHA256:%x", hash.Sum(nil)), nil
}

// slugCleanupFile removes the temporary slug file after upload.
// Mirrors cleanupSlugFile.
func slugCleanupFile(filePath string) {
	if filePath != "" {
		if err := os.Remove(filePath); err != nil {
			log.Printf("[WARN] Error cleaning up downloaded slug: %s (%s)", err, filePath)
		}
	}
}
