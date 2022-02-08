package heroku

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

// herokuApplication is a value type used to hold the details of an
// application. We use this for common storage of values needed for the
// heroku.App and heroku.TeamApp types
type herokuApplication struct {
	Name            string
	Region          string
	Space           string
	Stack           string
	InternalRouting bool
	GitURL          string
	WebURL          string
	TeamName        string
	Locked          bool
	Personal        bool
	Acm             bool
	ID              string
}

// type application is used to store all the details of a heroku app
type application struct {
	Id string // Id of the resource

	App        *herokuApplication // The heroku application
	Client     *heroku.Service    // Client to interact with the heroku API
	Vars       map[string]string  // Represents all vars on a heroku app.
	Buildpacks []string           // The application's buildpack names or URLs
	IsTeamApp  bool               // Is the application a team (organization) app
}

func resourceHerokuApp() *schema.Resource {
	return &schema.Resource{
		Create: switchHerokuAppCreate,
		Read:   resourceHerokuAppRead,
		Update: resourceHerokuAppUpdate,
		Delete: resourceHerokuAppDelete,
		Exists: resourceHerokuAppExists,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAppImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"space": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"stack": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"internal_routing": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"buildpacks": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"config_vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},

			"sensitive_config_vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
				Sensitive: true,
			},

			"all_config_vars": {
				Type:     schema.TypeMap,
				Computed: true,
				// These are marked Sensitive so that "sensitive_config_vars" do not
				// leak in the console/logs and also avoids unnecessary disclosure of
				// add-on secrets in logs.
				Sensitive: true,
			},

			"git_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"web_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"acm": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"heroku_hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"organization": {
				Type:     schema.TypeList,
				MinItems: 0,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							ForceNew:     true,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},

						"locked": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"personal": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
					},
				},
			},

			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuAppV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceHerokuAppStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func resourceHerokuAppImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*Config).Api

	app, err := client.AppInfo(context.TODO(), d.Id())
	if err != nil {
		return nil, err
	}

	// XXX Heroku's API treats app UUID's and names the same. This can cause
	// confusion as other parts of this provider assume the app NAME is the app
	// ID, as a lot of the Heroku API will accept BOTH. App ID's aren't very
	// easy to get, so it makes more sense to just use the name as much as possible.
	//
	// EDIT: (March 21, 2020) - The statement above causes issues for child resources
	// of heroku_app such as heroku_addon where the `app` attribute is often set to ForceNew.
	// As the app's name can change, its UUID does not. Therefore the heroku_app.id should be set to the UUID - DJ
	// Punting this change for now.
	d.SetId(app.ID)

	readErr := resourceHerokuAppRead(d, m)

	return []*schema.ResourceData{d}, readErr
}

func switchHerokuAppCreate(d *schema.ResourceData, meta interface{}) (err error) {
	if isTeamApp(d) {
		err = resourceHerokuTeamAppCreate(d, meta)
	} else {
		err = resourceHerokuAppCreate(d, meta)
	}
	if err == nil {
		config := meta.(*Config)
		time.Sleep(time.Duration(config.PostAppCreateDelay) * time.Second)
	}
	return
}

func resourceHerokuAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Build up our creation options
	opts := heroku.AppCreateOpts{}

	if v, ok := d.GetOk("name"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App name: %s", vs)
		opts.Name = &vs
	}
	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App region: %s", vs)
		opts.Region = &vs
	}
	if v, ok := d.GetOk("stack"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App stack: %s", vs)
		opts.Stack = &vs
	}

	log.Printf("[DEBUG] Creating Heroku app...")
	a, err := client.AppCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(a.ID)
	log.Printf("[INFO] App ID: %s", d.Id())

	if err := performAppPostCreateTasks(d, client); err != nil {
		return err
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuTeamAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Build up our creation options
	opts := heroku.TeamAppCreateOpts{}

	v := d.Get("organization").([]interface{})
	if len(v) > 1 {
		return fmt.Errorf("rrror Creating Heroku App: Only 1 Heroku Team (organization) is permitted")
	}
	newTeamApp := v[0].(map[string]interface{})

	if v := newTeamApp["name"]; v != nil {
		vs := v.(string)
		log.Printf("[DEBUG] Organization name: %s", vs)
		opts.Team = &vs
	}

	if v := newTeamApp["personal"]; v != nil {
		vs := v.(bool)
		log.Printf("[DEBUG] Organization Personal: %t", vs)
		opts.Personal = &vs
	}

	if v := newTeamApp["locked"]; v != nil {
		vs := v.(bool)
		log.Printf("[DEBUG] Organization locked: %t", vs)
		opts.Locked = &vs
	}

	if v := d.Get("name"); v != nil {
		vs := v.(string)
		log.Printf("[DEBUG] App name: %s", vs)
		opts.Name = &vs
	}
	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App region: %s", vs)
		opts.Region = &vs
	}
	if v, ok := d.GetOk("space"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App space: %s", vs)
		opts.Space = &vs
	}
	if v, ok := d.GetOk("stack"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App stack: %s", vs)
		opts.Stack = &vs
	}
	if v, ok := d.GetOk("internal_routing"); ok {
		vs := v.(bool)
		log.Printf("[DEBUG] App internal routing: %v", vs)
		opts.InternalRouting = &vs
	}

	log.Printf("[DEBUG] Creating Heroku app...")
	a, err := client.TeamAppCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(a.ID)
	log.Printf("[INFO] App ID: %s", d.Id())

	if err := performAppPostCreateTasks(d, client); err != nil {
		return err
	}

	return resourceHerokuAppRead(d, meta)
}

func setTeamDetails(d *schema.ResourceData, app *application) (err error) {
	err = d.Set("space", app.App.Space)

	teamAppDetails := map[string]interface{}{
		"name":   app.App.TeamName,
		"locked": app.App.Locked,

		// Platform API does not return this value so set state to resource schema value.
		"personal": d.Get("personal"),
	}
	err = d.Set("organization", []interface{}{teamAppDetails})

	return err
}

func setAppDetails(d *schema.ResourceData, app *application) (err error) {
	err = d.Set("name", app.App.Name)
	err = d.Set("stack", app.App.Stack)
	err = d.Set("internal_routing", app.App.InternalRouting)
	err = d.Set("region", app.App.Region)
	err = d.Set("git_url", app.App.GitURL)
	err = d.Set("web_url", app.App.WebURL)
	err = d.Set("acm", app.App.Acm)
	err = d.Set("uuid", app.App.ID)
	err = d.Set("heroku_hostname", fmt.Sprintf("%s.herokuapp.com", app.App.Name))

	return err
}

func resourceHerokuAppRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client := config.Api

	care := make(map[string]struct{})
	configVars := make(map[string]string)

	careSensitive := make(map[string]struct{})
	sensitiveConfigVars := make(map[string]string)

	// Only set the config_vars that we have set in the configuration.
	// The "all_config_vars" field has all of them.
	app, err := resourceHerokuAppRetrieve(d.Id(), client)
	if err != nil {
		return err
	}

	if c, ok := d.GetOk("config_vars"); ok {
		for k := range c.(map[string]interface{}) {
			care[k] = struct{}{}
		}
	}

	for k, v := range app.Vars {
		if _, ok := care[k]; ok {
			configVars[k] = v
		}
	}

	if s, ok := d.GetOk("sensitive_config_vars"); ok {
		for k := range s.(map[string]interface{}) {
			careSensitive[k] = struct{}{}
		}
	}

	for k, v := range app.Vars {
		if _, ok := careSensitive[k]; ok {
			sensitiveConfigVars[k] = v
		}
	}

	log.Printf("[LOG] Setting config vars: %s", configVars)
	if err := d.Set("config_vars", configVars); err != nil {
		log.Printf("[WARN] Error setting config vars: %s", err)
	}

	log.Printf("[LOG] Setting sensitive config vars: %s", sensitiveConfigVars)
	if err := d.Set("sensitive_config_vars", sensitiveConfigVars); err != nil {
		log.Printf("[WARN] Error setting sensitive config vars: %s", err)
	}

	// Set `all_config_vars` to empty map initially. Only set this attribute
	// if set_app_all_config_vars_in_state is `true`.
	d.Set("all_config_vars", map[string]string{})
	if config.SetAppAllConfigVarsInState {
		if err := d.Set("all_config_vars", app.Vars); err != nil {
			log.Printf("[WARN] Error setting all_config_vars: %s", err)
		}
	}

	buildpacksErr := d.Set("buildpacks", app.Buildpacks)
	if buildpacksErr != nil {
		return buildpacksErr
	}

	if app.IsTeamApp {
		orgErr := setTeamDetails(d, app)
		if orgErr != nil {
			return orgErr
		}
	}

	detailsErr := setAppDetails(d, app)
	if detailsErr != nil {
		return detailsErr
	}

	return nil
}

// resourceHerokuAppUpdate utilizes several unique API endpoints to update the app.
func resourceHerokuAppUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	opts := heroku.AppUpdateOpts{}

	// Make changes (if any) to the app itself.
	if d.HasChange("name") {
		v := d.Get("name").(string)
		opts.Name = &v
	}
	if d.HasChange("stack") {
		v := d.Get("stack").(string)
		opts.BuildStack = &v
	}

	updatedApp, err := client.AppUpdate(context.TODO(), d.Id(), opts)
	if err != nil {
		return err
	}
	d.Set("name", updatedApp.Name)

	// Make changes (if any) to the app's buildpack.
	if d.HasChange("buildpacks") {
		err := updateBuildpacks(d.Id(), client, d.Get("buildpacks").([]interface{}))
		if err != nil {
			return err
		}
	}

	// Make changes (if any) to the app's config vars.
	// Check if there are overlapping config vars and error out as precaution
	dupeErr := checkIfDupeConfigVars(d)
	if dupeErr != nil {
		return dupeErr
	}

	// If the config vars changed, then recalculate those
	var oldConfigVars, newConfigVars, oldSensitiveConfigVars, allOldVars, allNewVars,
		newSensitiveConfigVars map[string]interface{}

	log.Printf("[INFO] Does config_vars have change: *%#v", d.HasChange("config_vars"))
	if d.HasChange("config_vars") {
		o, n := d.GetChange("config_vars")
		if o == nil {
			o = make(map[string]interface{})
		}
		if n == nil {
			n = make(map[string]interface{})
		}

		oldConfigVars = o.(map[string]interface{})
		newConfigVars = n.(map[string]interface{})
	}

	log.Printf("[INFO] Does sensitive_config_vars have change: *%#v", d.HasChange("sensitive_config_vars"))
	if d.HasChange("sensitive_config_vars") {
		o, n := d.GetChange("sensitive_config_vars")
		if o == nil {
			o = make(map[string]interface{})
		}
		if n == nil {
			n = make(map[string]interface{})
		}

		oldSensitiveConfigVars = o.(map[string]interface{})
		newSensitiveConfigVars = n.(map[string]interface{})
	}

	// Merge the vars
	allOldVars = combineVars(oldConfigVars, oldSensitiveConfigVars)
	allNewVars = combineVars(newConfigVars, newSensitiveConfigVars)
	if err := updateConfigVars(d.Id(), client, allOldVars, allNewVars); err != nil {
		return err
	}

	// Make changes (if any) to the app's ACM.
	if d.HasChange("acm") {
		err := updateAcm(d.Id(), client, d.Get("acm").(bool))
		if err != nil {
			return err
		}
	}

	// Make changes (if any) to the app organization lock state.
	if d.HasChange("organization") {
		v := d.Get("organization").([]interface{})
		currentOrgAttrValues := v[0].(map[string]interface{})

		if v := currentOrgAttrValues["locked"]; v != nil {
			teamAppUpdateOpts := heroku.TeamAppUpdateLockedOpts{}
			vs := v.(bool)
			log.Printf("[DEBUG] Organization updated locked: %t", vs)
			teamAppUpdateOpts.Locked = vs

			_, lockUpdateErr := client.TeamAppUpdateLocked(context.TODO(), d.Id(), teamAppUpdateOpts)
			if lockUpdateErr != nil {
				return lockUpdateErr
			}
		}
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuAppDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting App: %s", d.Id())
	_, err := client.AppDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("error deleting App: %s", err)
	}

	return nil
}

func resourceHerokuAppExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	var err error
	client := meta.(*Config).Api

	if isTeamApp(d) {
		_, err = client.TeamAppInfo(context.TODO(), d.Id())
	} else {
		_, err = client.AppInfo(context.TODO(), d.Id())
	}
	if err != nil {
		// Make sure it's a missing app error.
		if herr, ok := err.(*url.Error).Err.(heroku.Error); ok && herr.ID == "not_found" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceHerokuAppRetrieve(id string, client *heroku.Service) (*application, error) {
	app := application{Id: id, Client: client, IsTeamApp: false}

	err := app.Update()

	if err != nil {
		return nil, fmt.Errorf("error retrieving app: %s", err)
	}

	return &app, nil
}

// Updates the application to have the latest from remote
func (a *application) Update() error {
	app, appGetErr := a.Client.AppInfo(context.TODO(), a.Id)
	if appGetErr != nil {
		return appGetErr
	}

	a.App = &herokuApplication{}
	a.App.Name = app.Name
	a.App.Region = app.Region.Name
	a.App.Stack = app.BuildStack.Name
	a.App.GitURL = app.GitURL
	a.App.WebURL = app.WebURL
	a.App.Acm = app.Acm
	a.App.ID = app.ID

	if app.InternalRouting != nil {
		a.App.InternalRouting = *app.InternalRouting
	}

	if app.Space != nil {
		a.App.Space = app.Space.Name
	}

	// If app is a team/org app, define additional values.
	if app.Organization != nil && app.Team != nil {
		// Set to true to control additional state actions downstream
		a.IsTeamApp = true

		// Need to do another API call to the /teams/apps endpoint to retrieve
		// additional info about a team app that isn't exposed through the /apps endpoint.
		teamApp, teamAppGetErr := a.Client.TeamAppInfo(context.TODO(), a.Id)
		if teamAppGetErr != nil {
			return teamAppGetErr
		}

		a.App.TeamName = teamApp.Team.Name
		a.App.Locked = teamApp.Locked
	}

	var errs []error
	var err error
	a.Buildpacks, err = retrieveBuildpacks(a.Id, a.Client)
	if err != nil {
		errs = append(errs, err)
	}

	a.Vars, err = retrieveConfigVars(a.Id, a.Client)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func isTeamApp(d *schema.ResourceData) bool {
	v := d.Get("organization").([]interface{})
	return len(v) > 0 && v[0] != nil
}

func retrieveBuildpacks(id string, client *heroku.Service) ([]string, error) {
	results, err := client.BuildpackInstallationList(context.TODO(), id, nil)

	if err != nil {
		return nil, err
	}

	buildpacks := make([]string, 0)
	for _, installation := range results {
		buildpacks = append(buildpacks, installation.Buildpack.Name)
	}

	return buildpacks, nil
}

func retrieveAcm(id string, client *heroku.Service) (bool, error) {
	result, err := client.AppInfo(context.TODO(), id)
	if err != nil {
		return false, err
	}
	return result.Acm, nil
}

func retrieveConfigVars(id string, client *heroku.Service) (map[string]string, error) {
	vars, err := client.ConfigVarInfoForApp(context.TODO(), id)

	if err != nil {
		return nil, err
	}

	nonNullVars := map[string]string{}
	for k, v := range vars {
		if v != nil {
			nonNullVars[k] = *v
		}
	}

	return nonNullVars, nil
}

// Updates the config vars for from an expanded configuration.
func updateConfigVars(id string, client *heroku.Service, o, n map[string]interface{}) error {
	vars := make(map[string]*string)

	for k := range o {
		vars[k] = nil
	}

	for k, v := range n {
		val := v.(string)
		vars[k] = &val
	}

	log.Printf("[INFO] Updating config vars: *%#v", vars)
	if _, err := client.ConfigVarUpdate(context.TODO(), id, vars); err != nil {
		return fmt.Errorf("Error updating config vars: %s", err)
	}

	releases, err := client.ReleaseList(
		context.TODO(),
		id,
		&heroku.ListRange{Descending: true, Field: "version", Max: 1},
	)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		return errors.New("no release found")
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"succeeded"},
		Refresh: releaseStateRefreshFunc(client, id, releases[0].ID),
		Timeout: 20 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for new release (%s) to succeed: %s", releases[0].ID, err)
	}

	return nil
}

func updateBuildpacks(id string, client *heroku.Service, v []interface{}) error {
	opts := heroku.BuildpackInstallationUpdateOpts{
		Updates: []struct {
			Buildpack string `json:"buildpack" url:"buildpack,key"`
		}{}}

	for _, buildpack := range v {
		opts.Updates = append(opts.Updates, struct {
			Buildpack string `json:"buildpack" url:"buildpack,key"`
		}{
			Buildpack: buildpack.(string),
		})
	}

	if _, err := client.BuildpackInstallationUpdate(context.TODO(), id, opts); err != nil {
		return fmt.Errorf("Error updating buildpacks: %s", err)
	}

	return nil
}

func updateAcm(id string, client *heroku.Service, enabled bool) error {
	if enabled {
		if _, err := client.AppEnableACM(context.TODO(), id); err != nil {
			return err
		}
	} else {
		if _, err := client.AppDisableACM(context.TODO(), id); err != nil {
			return err
		}
	}
	return nil
}

func combineVars(configVars, sensitiveConfigVars map[string]interface{}) map[string]interface{} {
	vars := make(map[string]interface{})

	for k, v := range configVars {
		vars[k] = v
	}

	for k, v := range sensitiveConfigVars {
		vars[k] = v
	}

	return vars
}

// performAppPostCreateTasks performs post-create tasks common to both org and non-org apps.
func performAppPostCreateTasks(d *schema.ResourceData, client *heroku.Service) error {
	// Check if there are overlapping config vars and error out as precaution
	dupeErr := checkIfDupeConfigVars(d)
	if dupeErr != nil {
		return dupeErr
	}

	// Create/Update/Delete Config Vars
	var configVars, sensitiveConfigVars, allConfigVars map[string]interface{}
	if v, ok := d.GetOk("config_vars"); ok {
		configVars = v.(map[string]interface{})
	}

	if v, ok := d.GetOk("sensitive_config_vars"); ok {
		sensitiveConfigVars = v.(map[string]interface{})
	}

	allConfigVars = combineVars(configVars, sensitiveConfigVars)

	if err := updateConfigVars(d.Id(), client, nil, allConfigVars); err != nil {
		return err
	}

	if v, ok := d.GetOk("buildpacks"); ok {
		if err := updateBuildpacks(d.Id(), client, v.([]interface{})); err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("acm"); ok {
		if _, ok := d.GetOk("organization"); !ok {
			log.Printf("You ask me to enable ACM for a non-organization app. This will most likely fail, due to the Heroku constraints (the app has to be scaled to Standard-1X - state of 28.01.2018)")
		}
		if err := updateAcm(d.Id(), client, v.(bool)); err != nil {
			return err
		}
	}

	return nil
}

func releaseStateRefreshFunc(client *heroku.Service, appID, releaseID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		release, err := client.ReleaseInfo(context.TODO(), appID, releaseID)

		if err != nil {
			return nil, "", err
		}

		// The type conversion here can be dropped when the vendored version of
		// heroku-go is updated.
		return (*heroku.Release)(release), release.Status, nil
	}
}

func checkIfDupeConfigVars(d *schema.ResourceData) error {
	log.Printf("[INFO] Checking for duplicate config vars")

	var dupes []string
	var configVars, sensitiveConfigVars map[string]interface{}
	if c, ok := d.GetOk("config_vars"); ok {
		configVars = c.(map[string]interface{})
	}

	if s, ok := d.GetOk("sensitive_config_vars"); ok {
		sensitiveConfigVars = s.(map[string]interface{})
	}

	if configVars != nil && sensitiveConfigVars != nil {
		for k := range configVars {
			if _, ok := sensitiveConfigVars[k]; ok {
				dupes = append(dupes, k)
			}
		}
	}

	log.Printf("[INFO] List of Duplicate config vars %s", dupes)

	if len(dupes) > 0 {
		return fmt.Errorf("[ERROR] Detected duplicate config vars: %s", dupes)
	}

	return nil
}

func resourceHerokuAppV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"space": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"stack": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"internal_routing": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"buildpacks": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"config_vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},

			"sensitive_config_vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
				Sensitive: true,
			},

			"all_config_vars": {
				Type:     schema.TypeMap,
				Computed: true,
				// These are marked Sensitive so that "sensitive_config_vars" do not
				// leak in the console/logs and also avoids unnecessary disclosure of
				// add-on secrets in logs.
				Sensitive: true,
			},

			"git_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"web_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"acm": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"heroku_hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"organization": {
				Type:     schema.TypeList,
				MinItems: 0,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							ForceNew:     true,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},

						"locked": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"personal": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
					},
				},
			},

			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuAppStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	appIdentifier := rawState["id"].(string)

	_, err := uuid.ParseUUID(appIdentifier)
	if err == nil {
		// id is already a valid UUID
		return rawState, nil
	}

	client := meta.(*Config).Api
	foundApp, err := client.AppInfo(ctx, appIdentifier)
	if err != nil {
		return nil, fmt.Errorf("resourceHerokuAppStateUpgradeV0 error retrieving app '%s': %w", appIdentifier, err)
	}
	rawState["id"] = foundApp.ID

	return rawState, nil
}
