package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                 = (*drainResource)(nil)
	_ resource.ResourceWithConfigure    = (*drainResource)(nil)
	_ resource.ResourceWithImportState  = (*drainResource)(nil)
	_ resource.ResourceWithUpgradeState = (*drainResource)(nil)
)

// NewDrainResource returns the framework implementation of the heroku_drain resource.
func NewDrainResource() resource.Resource {
	return &drainResource{}
}

type drainResource struct {
	config *Config
}

type drainResourceModel struct {
	ID           types.String `tfsdk:"id"`
	AppID        types.String `tfsdk:"app_id"`
	URL          types.String `tfsdk:"url"`
	SensitiveURL types.String `tfsdk:"sensitive_url"`
	Token        types.String `tfsdk:"token"`
}

// conflictsWithStringValidator is an inline implementation of the ConflictsWith
// validator (equivalent to stringvalidator.ConflictsWith from
// terraform-plugin-framework-validators), since that package is not vendored.
type conflictsWithStringValidator struct {
	expressions path.Expressions
}

func (v conflictsWithStringValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Conflicts with attributes: %v", v.expressions)
}

func (v conflictsWithStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v conflictsWithStringValidator) ValidateString(ctx context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
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

func (r *drainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_drain"
}

func (r *drainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				// Restore SDKv2 validation.IsUUID on app_id (lost in migration).
				Validators: []fwvalidator.String{
					uuidValidator(),
				},
			},
			"url": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					conflictsWithStringValidator{
						expressions: path.Expressions{path.MatchRoot("sensitive_url")},
					},
				},
			},
			"sensitive_url": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []fwvalidator.String{
					conflictsWithStringValidator{
						expressions: path.Expressions{path.MatchRoot("url")},
					},
				},
			},
			"token": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *drainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *drainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan drainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := plan.AppID.ValueString()

	// Check if app supports traditional drains (Cedar generation only).
	if err := validateAppSupportsTraditionalDrains(r.config.Api, appID); err != nil {
		resp.Diagnostics.AddError("App does not support traditional drains", err.Error())
		return
	}

	// Pick url from url or sensitive_url.
	var url string
	if !plan.URL.IsNull() && !plan.URL.IsUnknown() && plan.URL.ValueString() != "" {
		url = plan.URL.ValueString()
		log.Printf("[DEBUG] drain url: %s", url)
	}
	if !plan.SensitiveURL.IsNull() && !plan.SensitiveURL.IsUnknown() && plan.SensitiveURL.ValueString() != "" {
		url = plan.SensitiveURL.ValueString()
		log.Printf("[DEBUG] drain sensitive_url: %s", url)
	}

	log.Printf("[DEBUG] Drain create configuration: %#v, %#v", appID, url)

	// Retry loop mirroring SDKv2 resource.Retry(2*time.Minute).
	deadline := time.Now().Add(2 * time.Minute)
	var dr *heroku.LogDrain
	for {
		d, err := r.config.Api.LogDrainCreate(ctx, appID, heroku.LogDrainCreateOpts{URL: url})
		if err != nil {
			if strings.Contains(err.Error(), retryableError) {
				if time.Now().Before(deadline) {
					select {
					case <-ctx.Done():
						resp.Diagnostics.AddError("Context cancelled while creating drain", ctx.Err().Error())
						return
					case <-time.After(5 * time.Second):
					}
					continue
				}
				resp.Diagnostics.AddError("Timeout creating drain", err.Error())
				return
			}
			resp.Diagnostics.AddError("Error creating drain", err.Error())
			return
		}
		dr = d
		break
	}

	log.Printf("[INFO] Drain ID: %s", dr.ID)

	plan.ID = types.StringValue(dr.ID)

	// Read back current state.
	r.readDrainIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *drainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state drainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readDrainIntoModel(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op because every writable attribute is ForceNew.
// It writes the plan to state, preserving the id.
func (r *drainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan drainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state drainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *drainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state drainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting drain: %s", state.ID.ValueString())

	if _, err := r.config.Api.LogDrainDelete(ctx, state.AppID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting drain", fmt.Sprintf("Error deleting drain: %s", err))
		return
	}

	log.Printf("[INFO] Deleted drain: %s", state.ID.ValueString())
}

func (r *drainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	var app, id string
	var isSensitive bool

	switch len(parts) {
	case 2:
		app = parts[0]
		id = parts[1]
		isSensitive = false
	case 3:
		app = parts[0]
		id = parts[1]
		if parts[2] == "sensitive" {
			isSensitive = true
		} else {
			resp.Diagnostics.AddError(
				"Invalid import ID",
				fmt.Sprintf("To import a heroku_drain with a sensitive url, please use 'sensitive', not '%s'", parts[2]),
			)
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"The heroku_drain import ID should consist of 2 or 3 strings separated by a colon",
		)
		return
	}

	dr, err := r.config.Api.LogDrainInfo(ctx, app, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading drain during import", err.Error())
		return
	}

	foundApp, err := r.config.Api.AppInfo(ctx, app)
	if err != nil {
		resp.Diagnostics.AddError("Error reading app during import", err.Error())
		return
	}

	var state drainResourceModel
	state.ID = types.StringValue(dr.ID)
	state.AppID = types.StringValue(foundApp.ID)
	state.Token = types.StringValue(dr.Token)

	if isSensitive {
		state.SensitiveURL = types.StringValue(dr.URL)
		state.URL = types.StringNull()
	} else {
		state.URL = types.StringValue(dr.URL)
		state.SensitiveURL = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *drainResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	// v0 schema: uses "app" (not "app_id").
	v0Schema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"app": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Optional: true,
			},
			"sensitive_url": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"token": schema.StringAttribute{
				Computed: true,
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0Schema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				type drainV0Model struct {
					ID           types.String `tfsdk:"id"`
					App          types.String `tfsdk:"app"`
					URL          types.String `tfsdk:"url"`
					SensitiveURL types.String `tfsdk:"sensitive_url"`
					Token        types.String `tfsdk:"token"`
				}

				var prior drainV0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				appID, err := resolveAppToAppID(ctx, r.config, prior.App.ValueString(), "")
				if err != nil {
					resp.Diagnostics.AddError("Error resolving app to app_id during state upgrade", err.Error())
					return
				}

				upgraded := drainResourceModel{
					ID:           prior.ID,
					AppID:        types.StringValue(appID),
					URL:          prior.URL,
					SensitiveURL: prior.SensitiveURL,
					Token:        prior.Token,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

// readDrainIntoModel calls LogDrainInfo and populates the model, mirroring the
// SDKv2 resourceHerokuDrainRead GetOk logic: only set url or sensitive_url
// depending on which is non-null in state.
func (r *drainResource) readDrainIntoModel(ctx context.Context, m *drainResourceModel, diags *diag.Diagnostics) {
	dr, err := r.config.Api.LogDrainInfo(ctx, m.AppID.ValueString(), m.ID.ValueString())
	if err != nil {
		diags.AddError("Error retrieving drain", fmt.Sprintf("Error retrieving drain: %s", err))
		return
	}

	m.Token = types.StringValue(dr.Token)

	if !m.URL.IsNull() {
		m.URL = types.StringValue(dr.URL)
	}
	if !m.SensitiveURL.IsNull() {
		m.SensitiveURL = types.StringValue(dr.URL)
	}
}
