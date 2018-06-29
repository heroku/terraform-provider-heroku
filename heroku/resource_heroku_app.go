package heroku

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/cyberdelia/heroku-go/v3"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// herokuApplication is a value type used to hold the details of an
// application. We use this for common storage of values needed for the
// heroku.App and heroku.TeamApp types
type herokuApplication struct {
	Name     string
	Region   string
	Space    string
	Stack    string
	GitURL   string
	WebURL   string
	TeamName string
	Locked   bool
	Acm      bool
}

// type application is used to store all the details of a heroku app
type application struct {
	Id string // Id of the resource

	App        *herokuApplication // The heroku application
	Client     *heroku.Service    // Client to interact with the heroku API
	Vars       map[string]string  // The vars on the application
	Buildpacks []string           // The application's buildpack names or URLs
	Team       bool               // is the application team/organization
}

// Updates the application to have the latest from remote
func (a *application) Update() error {
	var errs []error
	var err error

	app, err := a.Client.AppInfo(context.TODO(), a.Id)
	if err != nil {
		errs = append(errs, err)
	} else {
		a.App = &herokuApplication{}
		a.App.Name = app.Name
		a.App.Region = app.Region.Name
		a.App.Stack = app.BuildStack.Name
		a.App.GitURL = app.GitURL
		a.App.WebURL = app.WebURL
		a.App.Acm = app.Acm
		if app.Space != nil {
			a.App.Space = app.Space.Name
		}

		if app.Team != nil {
			a.App.TeamName = app.Team.Name
		} else if app.Organization != nil {
			a.App.TeamName = app.Organization.Name
		} else {
			log.Println("[DEBUG] Something is wrong - didn't get information about organization name, while the app is marked as being so")
		}
	}

	if app.Team != nil {
		a.App.Locked, err = retrieveTeamLockState(a.Id, app.TeamName, a.Client)
		if err != nil {
			errs = append(errs, err)
		}
	}

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

			"buildpacks": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"config_vars": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},

			"all_config_vars": {
				Type:     schema.TypeMap,
				Computed: true,
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
			},

			"heroku_hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"organization": {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: "team",
				Deprecated:    "Heroku has deprecated organizations. Use team instead.",
				MaxItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"locked": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"personal": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"team": {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: "organization",
				MaxItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"locked": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"personal": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func isTeamApp(d *schema.ResourceData) bool {
	// The "team" and "organization" attributes are synonyms. Organizations have been deprecated
	// by heroku and should no longer be used.
	// See https://devcenter.heroku.com/changelog-items/1132
	org := d.Get("organization").([]interface{})
	team := d.Get("team").([]interface{})
	return ((len(org) > 0 && org[0] != nil) || (len(team) > 0 && team[0] != nil))
}

func resourceHerokuAppImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*heroku.Service)

	app, err := client.AppInfo(context.TODO(), d.Id())
	if err != nil {
		return nil, err
	}

	// @joestump 06/29/2018 Heroku has deprecated organizations. This will set the "name" attribute
	// on the "team" map to the team name if it exists with organization name as fallback.
	// See https://github.com/cyberdelia/heroku-go/pull/27#issuecomment-401399969
	teamName := ""
	if app.Team != nil {
		teamName = app.Team.Name
	} else if app.Organization {
		teamName = app.Organization.Name
	}

	if teamName != "" {
		d.Set("team", []map[string]interface{}{
			{"name": teamName},
		})
	}

	// XXX Heroku's API treats app UUID's and names the same. This can cause
	// confusion as other parts of this provider assume the app NAME is the app
	// ID, as a lot of the Heroku API will accept BOTH. App ID's aren't very
	// easy to get, so it makes more sense to just use the name as much as possible.
	d.SetId(app.Name)
	d.Set("acm", app.Acm)

	return []*schema.ResourceData{d}, nil
}

func switchHerokuAppCreate(d *schema.ResourceData, meta interface{}) error {
	if isTeamApp(d) {
		return resourceHerokuTeamAppCreate(d, meta)
	}

	return resourceHerokuAppCreate(d, meta)
}

func resourceHerokuAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

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

	d.SetId(a.Name)
	log.Printf("[INFO] App ID: %s", d.Id())

	if err := performAppPostCreateTasks(d, client); err != nil {
		return err
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuTeamAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	opts := heroku.TeamAppCreateOpts{}

	if v, ok := d.GetOk("team"); ok {
		v := d.Get("team").([]interface{})
	} else if v, ok := d.GetOk("organization"); ok {
		v := d.Get("organization").([]interface{})
	}

	teamDetails := v[0].(map[string]interface{})

	if v := teamDetails["name"]; v != nil {
		vs := v.(string)
		log.Printf("[DEBUG] Team Name: %s", vs)
		opts.Team = &vs
	}

	if v := teamDetails["personal"]; v != nil {
		vs := v.(bool)
		log.Printf("[DEBUG] Team Personal: %t", vs)
		opts.Personal = &vs
	}

	if v := teamDetails["locked"]; v != nil {
		vs := v.(bool)
		log.Printf("[DEBUG] Team locked: %t", vs)
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

	log.Printf("[DEBUG] Creating Heroku Team App...")
	a, err := client.TeamAppCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(a.Name)
	log.Printf("[INFO] App ID: %s", d.Id())

	if err := performAppPostCreateTasks(d, client); err != nil {
		return err
	}

	return resourceHerokuAppRead(d, meta)
}

func setTeamDetails(d *schema.ResourceData, app *application) (err error) {
	d.Set("space", app.App.Space)

	teamDetails := map[string]interface{}{
		"name":     app.App.TeamName,
		"locked":   app.App.Locked,
		"personal": false,
	}
	err = d.Set("team", []interface{}{orgDetails})
	return err
}

func setAppDetails(d *schema.ResourceData, app *application) (err error) {
	d.Set("name", app.App.Name)
	d.Set("stack", app.App.Stack)
	d.Set("region", app.App.Region)
	d.Set("git_url", app.App.GitURL)
	d.Set("web_url", app.App.WebURL)
	d.Set("acm", app.App.Acm)
	d.Set("heroku_hostname", fmt.Sprintf("%s.herokuapp.com", app.App.Name))
	return err
}

func resourceHerokuAppRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	care := make(map[string]struct{})
	configVars := make(map[string]string)

	// Only track buildpacks when set in the configuration.
	_, buildpacksConfigured := d.GetOk("buildpacks")

	teamApp := isTeamApp(d)

	// Only set the config_vars that we have set in the configuration.
	// The "all_config_vars" field has all of them.
	app, err := resourceHerokuAppRetrieve(d.Id(), teamApp, client)
	if err != nil {
		return err
	}

	for _, v := range d.Get("config_vars").([]interface{}) {
		// Protect against panic on type cast for a nil-length array or map
		n, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		for k := range n {
			care[k] = struct{}{}
		}
	}

	for k, v := range app.Vars {
		if _, ok := care[k]; ok {
			configVars[k] = v
		}
	}

	var configVarsValue []map[string]string
	if len(configVars) > 0 {
		configVarsValue = []map[string]string{configVars}
	}

	if buildpacksConfigured {
		d.Set("buildpacks", app.Buildpacks)
	}

	if err := d.Set("config_vars", configVarsValue); err != nil {
		log.Printf("[WARN] Error setting config vars: %s", err)
	}

	if err := d.Set("all_config_vars", app.Vars); err != nil {
		log.Printf("[WARN] Error setting all_config_vars: %s", err)
	}

	if teamApp {
		setTeamDetails(d, app)
	}

	setAppDetails(d, app)

	return nil
}

func resourceHerokuAppUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)
	opts := heroku.AppUpdateOpts{}

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
	d.SetId(updatedApp.Name)

	if d.HasChange("buildpacks") {
		err := updateBuildpacks(d.Id(), client, d.Get("buildpacks").([]interface{}))
		if err != nil {
			return err
		}
	}

	// If the config vars changed, then recalculate those
	if d.HasChange("config_vars") {
		o, n := d.GetChange("config_vars")
		if o == nil {
			o = []interface{}{}
		}
		if n == nil {
			n = []interface{}{}
		}

		err := updateConfigVars(
			d.Id(), client, o.([]interface{}), n.([]interface{}))
		if err != nil {
			return err
		}

		releases, err := client.ReleaseList(
			context.TODO(),
			d.Id(),
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
			Refresh: releaseStateRefreshFunc(client, d.Id(), releases[0].ID),
			Timeout: 20 * time.Minute,
		}

		if _, err := stateConf.WaitForState(); err != nil {
			return fmt.Errorf("Error waiting for new release (%s) to succeed: %s", releases[0].ID, err)
		}
	}

	if d.HasChange("acm") {
		err := updateAcm(d.Id(), client, d.Get("acm").(bool))
		if err != nil {
			return err
		}
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuAppDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting App: %s", d.Id())
	_, err := client.AppDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting App: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuAppExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	var err error
	client := meta.(*heroku.Service)

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

func resourceHerokuAppRetrieve(id string, team bool, client *heroku.Service) (*application, error) {
	app := application{Id: id, Client: client, Team: organization}

	err := app.Update()

	if err != nil {
		return nil, fmt.Errorf("Error retrieving app: %s", err)
	}

	return &app, nil
}

func retrieveTeamLockState(id, team string, client *heroku.Service) (bool, error) {
	app, err := client.TeamAppInfo(context.TODO(), id)
	if err != nil {
		return false, err
	}

	return app.Locked, nil
}

func retrieveBuildpacks(id string, client *heroku.Service) ([]string, error) {
	results, err := client.BuildpackInstallationList(context.TODO(), id, nil)

	if err != nil {
		return nil, err
	}

	buildpacks := []string{}
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
func updateConfigVars(
	id string,
	client *heroku.Service,
	o []interface{},
	n []interface{}) error {
	vars := make(map[string]*string)

	for _, v := range o {
		if v != nil {
			for k := range v.(map[string]interface{}) {
				vars[k] = nil
			}
		}
	}
	for _, v := range n {
		if v != nil {
			for k, v := range v.(map[string]interface{}) {
				val := v.(string)
				vars[k] = &val
			}
		}
	}

	log.Printf("[INFO] Updating config vars: *%#v", vars)
	if _, err := client.ConfigVarUpdate(context.TODO(), id, vars); err != nil {
		return fmt.Errorf("Error updating config vars: %s", err)
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

// performAppPostCreateTasks performs post-create tasks common to both team/org and non-org apps.
func performAppPostCreateTasks(d *schema.ResourceData, client *heroku.Service) error {
	if v, ok := d.GetOk("config_vars"); ok {
		if err := updateConfigVars(d.Id(), client, nil, v.([]interface{})); err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("buildpacks"); ok {
		if err := updateBuildpacks(d.Id(), client, v.([]interface{})); err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("acm"); ok {
		if _, ok := d.GetOk("team"); !ok {
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
