package heroku

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
	"time"
)

func resourceHerokuAppRelease() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAppReleaseCreate,
		Read:   resourceHerokuAppReleaseRead,
		Update: resourceHerokuAppReleaseUpdate,
		Delete: resourceHerokuAppReleaseDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAppReleaseImport,
		},

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"slug_id": { // An existing Heroku release cannot be updated so ForceNew is required
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceHerokuAppReleaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.ReleaseCreateOpts{}

	appName := getAppName(d)

	if v, ok := d.GetOk("slug_id"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] Slug Id: %s", vs)
		opts.Slug = vs
	}

	if v, ok := d.GetOk("description"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] description: %s", vs)
		opts.Description = &vs
	}

	log.Printf("[DEBUG] Creating a new release on app: [%s]", appName)
	newRelease, err := client.ReleaseCreate(context.TODO(), appName, opts)

	if err != nil {
		return err
	}

	log.Printf("[INFO] New release ID: %s", newRelease.ID)
	log.Printf("[INFO] Begin Checking if new Release %s is successful", newRelease.ID)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"succeeded"},
		Refresh: releaseStateRefreshFunc(client, appName, newRelease.ID),
		Timeout: 20 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("[ERROR] Error waiting for new release (%s) to succeed: %s", newRelease.ID, err)
	}

	// Set the ID after the release is successful
	d.SetId(newRelease.ID)

	return resourceHerokuAppReleaseRead(d, meta)
}

func resourceHerokuAppReleaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appName := getAppName(d)

	appRelease, err := client.ReleaseInfo(context.TODO(), appName, d.Id())

	if err != nil {
		return fmt.Errorf("[ERROR] error retrieving app release: %s", err)
	}

	d.Set("app", appRelease.App.Name)
	d.Set("slug_id", appRelease.Slug.ID)
	d.Set("description", appRelease.Description)

	return nil
}

// resourceHerokuAppReleaseUpdate will be a no-op method as there is no UPDATE endpoint for the release resource
// in the Heroku Platform APIs.
func resourceHerokuAppReleaseUpdate(d *schema.ResourceData, meta interface{}) error {
	// Detect if [description] attribute changed but not [slug_id]. If such is the case, output error.
	// If both attributes changed, a new release will be created since [slug_id] is set to ForceNew.

	if !d.HasChange("slug_id") && d.HasChange("description") {
		return errors.New("you cannot update an existing release's description. Please create a new release instead")
	}

	return nil
}

// resourceHerokuAppReleaseDelete will be a no-op method as there is no DELETE endpoint for the release resource
// in the Heroku Platform APIs.
func resourceHerokuAppReleaseDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] There is no DELETE for releease resource so this is a no-op. Resource will be removed from state.")
	return nil
}

func resourceHerokuAppReleaseImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// The import function will import the current release for an app.
	// There doesn't seem to be a compelling reason for someone to import a legacy release on an application.
	client := meta.(*Config).Api

	appName := d.Id()

	log.Printf("[INFO] Importing Release for App [%s]", appName)

	appReleases, err := client.ReleaseList(context.Background(), appName, &heroku.ListRange{Descending: true, Field: "version", Max: 1})
	appRelease := appReleases[0]

	if err != nil {
		return nil, err
	}

	// It isn't likely for the last release to not be 'current', but adding the check below just to be sure
	if !appRelease.Current {
		return nil, fmt.Errorf("[ERROR] The latest release for app [%s] is not current for some reason", appName)
	}

	d.SetId(appRelease.ID)
	d.Set("app", appRelease.App.Name)
	d.Set("slug_id", appRelease.Slug.ID)
	d.Set("description", appRelease.Description)

	return []*schema.ResourceData{d}, nil
}
