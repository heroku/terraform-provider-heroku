package heroku

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	heroku "github.com/heroku/heroku-go/v6"
)

// This file holds the SDKv2-free application helpers shared by the
// terraform-plugin-framework resources and data sources (heroku_app,
// heroku_app_release, heroku_app_config_association, heroku_slug,
// heroku_domain, and the heroku_app data source). They were extracted from the
// original SDKv2 resource_heroku_app.go when that resource was migrated to the
// framework, so the framework code no longer depends on
// terraform-plugin-sdk/v2.

// herokuApplication is a value type used to hold the details of an
// application. We use this for common storage of values needed for the
// heroku.App and heroku.TeamApp types.
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

// application is used to store all the details of a heroku app.
type application struct {
	Id string // Id of the resource

	App        *herokuApplication // The heroku application
	Client     *heroku.Service    // Client to interact with the heroku API
	Vars       map[string]string  // Represents all vars on a heroku app.
	Buildpacks []string           // The application's buildpack names or URLs
	IsTeamApp  bool               // Is the application a team (organization) app
	Generation string             // The generation of the app platform (cedar/fir)
}

func resourceHerokuAppRetrieve(id string, client *heroku.Service) (*application, error) {
	app := application{Id: id, Client: client, IsTeamApp: false}

	err := app.Update()

	if err != nil {
		return nil, fmt.Errorf("error retrieving app: %s", err)
	}

	return &app, nil
}

// Update refreshes the application with the latest values from the remote.
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
	if app.WebURL != nil {
		a.App.WebURL = *app.WebURL
	}
	a.App.Acm = app.Acm
	a.App.ID = app.ID

	if app.InternalRouting != nil {
		a.App.InternalRouting = *app.InternalRouting
	}

	if app.Space != nil {
		a.App.Space = app.Space.Name
	}

	// Determine generation from app's generation field (available in AppInfo response)
	if app.Generation.Name != "" {
		a.Generation = app.Generation.Name
	} else {
		// Default to cedar if generation is not specified
		a.Generation = "cedar"
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

	// Only retrieve buildpacks for apps that support traditional buildpacks
	if IsFeatureSupported(a.Generation, "app", "buildpacks") {
		a.Buildpacks, err = retrieveBuildpacks(a.Id, a.Client)
		if err != nil {
			errs = append(errs, err)
		}
	} else {
		// CNB apps don't have traditional buildpacks
		log.Printf("[DEBUG] App %s uses generation %s which doesn't support traditional buildpacks", a.Id, a.Generation)
		a.Buildpacks = []string{}
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

// isCNBError checks if an error is related to Cloud Native Buildpacks.
func isCNBError(err error) bool {
	if err == nil {
		return false
	}
	errorMessage := err.Error()
	return strings.Contains(errorMessage, "Cloud Native Buildpacks") ||
		strings.Contains(errorMessage, "project.toml")
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

// updateConfigVars updates the config vars from an expanded configuration and
// waits for the resulting release to succeed.
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

	if err := waitForReleaseSucceeded(context.TODO(), client, id, releases[0].ID, 20*time.Minute); err != nil {
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

// waitForReleaseSucceeded polls the given release until it reaches the
// "succeeded" status, honouring context cancellation and the supplied timeout.
// It replaces the SDKv2 resource.StateChangeConf + releaseStateRefreshFunc
// polling used before the migration to terraform-plugin-framework. A status
// other than "pending"/"succeeded" is treated as an unexpected state, matching
// the previous StateChangeConf Pending/Target semantics.
func waitForReleaseSucceeded(ctx context.Context, client *heroku.Service, appID, releaseID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		release, err := client.ReleaseInfo(ctx, appID, releaseID)
		if err != nil {
			return err
		}

		switch release.Status {
		case "succeeded":
			return nil
		case "pending":
			// Still releasing; keep polling.
		default:
			return fmt.Errorf("release %s entered unexpected status %q", releaseID, release.Status)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout while waiting for release %s to succeed", releaseID)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}
