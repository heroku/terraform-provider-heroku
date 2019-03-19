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

func resourceHerokuAppConfigAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAppConfigAssociationCreate,
		Read:   resourceHerokuAppConfigAssociationRead,
		Update: resourceHerokuAppConfigAssociationUpdate,
		Delete: resourceHerokuAppConfigAssociationDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAppConfigAssociationImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"sensitive_vars": {
				Type:      schema.TypeMap,
				Sensitive: true,
				Optional:  true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
			},
		},
	}
}

// As config var sensitivity is not a built-in Heroku distinction, it will not be possible to import this resource.
func resourceHerokuAppConfigAssociationImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	noImportErr := fmt.Errorf("it is not possible to import heroku_app_config_association since there are no remote resources")

	return nil, noImportErr
}

func resourceHerokuAppConfigAssociationCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	appId := getAppId(d)
	configVars := getVars(d)
	sensitiveConfigVars := getSensitiveVars(d)

	// Check for duplicates
	dupeErr := duplicateChecker(configVars, sensitiveConfigVars)
	if dupeErr != nil {
		return dupeErr
	}

	// Combine Both Variables
	combinedVars := mergeVars(configVars, sensitiveConfigVars)

	// Update vars on the app
	if err := updateVars(appId, client, nil, combinedVars); err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("config:%s", appId))
	setErr := d.Set("app_id", appId)
	if setErr != nil {
		return setErr
	}

	return resourceHerokuAppConfigAssociationRead(d, m)
}

func resourceHerokuAppConfigAssociationRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	appId := getAppId(d)

	vettedVars := make(map[string]string)
	vettedSensitiveVars := make(map[string]string)
	vars := getVars(d)
	sensitiveVars := getSensitiveVars(d)

	remoteAppVars, remoteAppGetErr := retrieveConfigVars(appId, client)
	if remoteAppGetErr != nil {
		return remoteAppGetErr
	}

	// Verify through each vars and sensitiveVars by checking each key, value pair against what was set romotely
	for k := range vars {
		vettedVars[k] = remoteAppVars[k]
	}

	for k := range sensitiveVars {
		vettedSensitiveVars[k] = remoteAppVars[k]
	}

	if err := d.Set("vars", vettedVars); err != nil {
		log.Printf("[WARN] Error setting vars: %s", err)
	}
	if err := d.Set("sensitive_vars", vettedSensitiveVars); err != nil {
		log.Printf("[WARN] Error setting sensitive vars: %s", err)
	}

	return nil
}

func resourceHerokuAppConfigAssociationUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api
	appId := getAppId(d)

	var oldVars, newVars, oldSensitiveVars, newSensitiveVars, allOldVars, allNewVars map[string]interface{}
	oldVars, newVars = getVarDiff(d, "vars")
	oldSensitiveVars, newSensitiveVars = getVarDiff(d, "sensitive_vars")

	// Merge the vars
	allOldVars = mergeVars(oldVars, oldSensitiveVars)
	allNewVars = mergeVars(newVars, newSensitiveVars)

	// Update vars on the app
	if err := updateVars(appId, client, allOldVars, allNewVars); err != nil {
		return err
	}

	return resourceHerokuAppConfigAssociationRead(d, m)
}

func resourceHerokuAppConfigAssociationDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api
	appId := getAppId(d)

	vars := getVars(d)
	sensitiveVars := getSensitiveVars(d)
	allVars := mergeVars(vars, sensitiveVars)

	// Essentially execute an update to delete all the vars listed in the schema only
	if err := updateVars(appId, client, allVars, nil); err != nil {
		return err
	}

	// Remove resource from state
	d.SetId("")

	return nil
}

func mergeVars(configVars, sensitiveVars map[string]interface{}) map[string]interface{} {
	combined := make(map[string]interface{})

	for k, v := range configVars {
		if v != nil {
			combined[k] = v
		}
	}

	for k, v := range sensitiveVars {
		if v != nil {
			combined[k] = v
		}
	}

	return combined
}

func updateVars(id string, client *heroku.Service, o map[string]interface{}, n map[string]interface{}) error {
	vars := make(map[string]*string)

	for k, v := range o {
		if v != nil {
			vars[k] = nil
		}
	}

	for k, v := range n {
		if v != nil {
			val := v.(string)
			vars[k] = &val
		}
	}

	log.Printf("[INFO] Updating config vars: *%#v", vars)
	if _, err := client.ConfigVarUpdate(context.TODO(), id, vars); err != nil {
		return fmt.Errorf("error updating config vars: %s", err)
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
		return fmt.Errorf("error waiting for new release (%s) to succeed: %s", releases[0].ID, err)
	}

	return nil
}

func getVarDiff(d *schema.ResourceData, key string) (old, new map[string]interface{}) {
	log.Printf("[INFO] Does %s have change: *%#v", key, d.HasChange(key))
	if d.HasChange(key) {
		o, n := d.GetChange(key)
		if o == nil {
			o = map[string]interface{}{}
		}
		if n == nil {
			n = map[string]interface{}{}
		}

		old = o.(map[string]interface{})
		new = n.(map[string]interface{})
	}

	return old, new
}

// getVars extracts the vars attribute generically from a Heroku resource.
func getVars(d *schema.ResourceData) map[string]interface{} {
	var vars map[string]interface{}
	if v, ok := d.GetOk("vars"); ok {
		vs := v.(map[string]interface{})
		log.Printf("[DEBUG] vars: %s", vs)
		vars = vs
	}

	return vars
}

// getVars extracts the vars attribute generically from a Heroku resource.
func getSensitiveVars(d *schema.ResourceData) map[string]interface{} {
	var sensitiveVars map[string]interface{}
	if v, ok := d.GetOk("sensitive_vars"); ok {
		vs := v.(map[string]interface{})
		log.Printf("[DEBUG] sensitive vars: %s", vs)
		sensitiveVars = vs
	}

	return sensitiveVars
}
