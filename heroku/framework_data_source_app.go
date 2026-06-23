package heroku

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	heroku "github.com/heroku/heroku-go/v6"
)

// Ensure interface compliance at compile time.
var (
	_ datasource.DataSource              = (*appDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*appDataSource)(nil)
)

// NewAppDataSource returns the framework implementation of the heroku_app data source.
func NewAppDataSource() datasource.DataSource {
	return &appDataSource{}
}

type appDataSource struct {
	config *Config
}

// appDataSourceOrganizationModel mirrors the nested "organization" block from
// the SDK data source schema.
type appDataSourceOrganizationModel struct {
	Name     types.String `tfsdk:"name"`
	Locked   types.Bool   `tfsdk:"locked"`
	Personal types.Bool   `tfsdk:"personal"`
}

// appDataSourceModel holds all state fields for the heroku_app data source.
type appDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Space           types.String `tfsdk:"space"`
	Region          types.String `tfsdk:"region"`
	Stack           types.String `tfsdk:"stack"`
	Generation      types.String `tfsdk:"generation"`
	InternalRouting types.Bool   `tfsdk:"internal_routing"`
	Buildpacks      types.List   `tfsdk:"buildpacks"`
	ConfigVars      types.Map    `tfsdk:"config_vars"`
	AllConfigVars   types.Map    `tfsdk:"all_config_vars"`
	GitURL          types.String `tfsdk:"git_url"`
	WebURL          types.String `tfsdk:"web_url"`
	Acm             types.Bool   `tfsdk:"acm"`
	HerokuHostname  types.String `tfsdk:"heroku_hostname"`
	LastReleaseID   types.String `tfsdk:"last_release_id"`
	LastSlugID      types.String `tfsdk:"last_slug_id"`
	Organization    types.List   `tfsdk:"organization"`
	UUID            types.String `tfsdk:"uuid"`
}

func (d *appDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *appDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"space": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				Computed: true,
			},
			"stack": schema.StringAttribute{
				Computed: true,
			},
			"generation": schema.StringAttribute{
				Computed:    true,
				Description: "Generation of the app platform (cedar or fir)",
			},
			"internal_routing": schema.BoolAttribute{
				Computed: true,
			},
			"buildpacks": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"config_vars": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"all_config_vars": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
			"git_url": schema.StringAttribute{
				Computed: true,
			},
			"web_url": schema.StringAttribute{
				Computed: true,
			},
			"acm": schema.BoolAttribute{
				Computed: true,
			},
			"heroku_hostname": schema.StringAttribute{
				Computed: true,
			},
			"last_release_id": schema.StringAttribute{
				Computed: true,
			},
			"last_slug_id": schema.StringAttribute{
				Computed: true,
			},
			"organization": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"locked": schema.BoolAttribute{
							Computed: true,
						},
						"personal": schema.BoolAttribute{
							Computed: true,
						},
					},
				},
			},
			"uuid": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *appDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.config = config
}

func (d *appDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config appDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	app, err := resourceHerokuAppRetrieve(name, d.config.Api)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to retrieve heroku_app %q", name),
			err.Error(),
		)
		return
	}

	// --- id / uuid (mirrors d.SetId(app.App.ID) + d.Set("uuid", app.App.ID)) ---
	config.ID = types.StringValue(app.App.ID)
	config.UUID = types.StringValue(app.App.ID)

	// --- setAppDetails equivalent ---
	config.Name = types.StringValue(app.App.Name)
	config.Stack = types.StringValue(app.App.Stack)
	config.InternalRouting = types.BoolValue(app.App.InternalRouting)
	config.Region = types.StringValue(app.App.Region)
	config.GitURL = types.StringValue(app.App.GitURL)
	config.WebURL = types.StringValue(app.App.WebURL)
	config.Acm = types.BoolValue(app.App.Acm)
	config.HerokuHostname = types.StringValue(
		strings.TrimPrefix(
			strings.TrimPrefix(
				strings.TrimSuffix(app.App.WebURL, "/"),
				"https://"),
			"http://"),
	)

	// --- generation ---
	config.Generation = types.StringValue(app.Generation)

	// --- setTeamDetails equivalent ---
	// space is set for team apps; for non-team apps it remains empty string
	config.Space = types.StringValue(app.App.Space)

	orgAttrTypes := map[string]attr.Type{
		"name":     types.StringType,
		"locked":   types.BoolType,
		"personal": types.BoolType,
	}

	if app.IsTeamApp {
		orgObj, diags := types.ObjectValue(
			orgAttrTypes,
			map[string]attr.Value{
				"name":   types.StringValue(app.App.TeamName),
				"locked": types.BoolValue(app.App.Locked),
				// The Platform API does not return the "personal" flag for an existing
				// team app (only settable at creation time). Mirror SDKv2 behaviour and
				// default to false when not available.
				"personal": types.BoolValue(false),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		orgList, diags := types.ListValue(
			types.ObjectType{AttrTypes: orgAttrTypes},
			[]attr.Value{orgObj},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Organization = orgList
	} else {
		config.Organization = types.ListValueMust(
			types.ObjectType{AttrTypes: orgAttrTypes},
			[]attr.Value{},
		)
	}

	// --- buildpacks (already populated by resourceHerokuAppRetrieve → app.Update) ---
	buildpackVals := make([]attr.Value, len(app.Buildpacks))
	for i, bp := range app.Buildpacks {
		buildpackVals[i] = types.StringValue(bp)
	}
	buildpackList, diags := types.ListValue(types.StringType, buildpackVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Buildpacks = buildpackList

	// --- config_vars: all vars retrieved by resourceHerokuAppRetrieve ---
	configVarElems := make(map[string]attr.Value, len(app.Vars))
	for k, v := range app.Vars {
		configVarElems[k] = types.StringValue(v)
	}
	configVarsMap, diags := types.MapValue(types.StringType, configVarElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.ConfigVars = configVarsMap

	// --- all_config_vars: same set of vars (data source exposes everything) ---
	allConfigVarsMap, diags := types.MapValue(types.StringType, configVarElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.AllConfigVars = allConfigVarsMap

	// --- last_release_id / last_slug_id ---
	// Fetch releases sorted descending by version and find the first succeeded one,
	// mirroring the SDKv2 dataSourceHerokuAppRead logic exactly.
	releaseRange := heroku.ListRange{
		Field:      "version",
		Max:        200,
		Descending: true,
	}
	releases, err := d.config.Api.ReleaseList(ctx, app.App.ID, &releaseRange)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to fetch releases for app %q", name),
			err.Error(),
		)
		return
	}

	config.LastReleaseID = types.StringValue("")
	config.LastSlugID = types.StringValue("")
	for _, r := range releases {
		if r.Status == "succeeded" {
			config.LastReleaseID = types.StringValue(r.ID)
			if r.Slug != nil {
				config.LastSlugID = types.StringValue(r.Slug.ID)
			}
			break
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
