package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"strconv"
	"time"
)

func resourceHerokuConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuConfigCreate,
		Read:   resourceHerokuConfigRead,
		Update: resourceHerokuConfigUpdate,
		Delete: resourceHerokuConfigDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuConfigImport,
		},

		Schema: map[string]*schema.Schema{
			"vars": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"sensitive_vars": {
				Type:      schema.TypeMap,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
			},
		},
	}
}

// It will not be possible to import this resource as  heroku_config does not interact with any remote resources.
// Therefore, this function will notify user of this inability.
func resourceHerokuConfigImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	noImportErr := fmt.Errorf("it is not possible to import heroku_config since there are no remote resources" +
		" associated with heroku_config")

	return nil, noImportErr
}

func resourceHerokuConfigCreate(d *schema.ResourceData, m interface{}) error {
	var vars, sensitiveVars map[string]interface{}

	if v, ok := d.GetOk("vars"); ok {
		vs := v.(map[string]interface{})
		log.Printf("[DEBUG] vars: %v", vs)
		vars = vs
	}

	if v, ok := d.GetOk("sensitive_vars"); ok {
		vs := v.(map[string]interface{})
		log.Printf("[DEBUG] sensitive vars: %v", vs)
		sensitiveVars = vs
	}

	// Check for duplicate values. If there are duplicates, error out as a preventative measure
	dupeErr := duplicateVarsChecker(vars, sensitiveVars)
	if dupeErr != nil {
		return dupeErr
	}

	// Set the ID to be name + epoch time for uniqueness
	epochTime := time.Now().Unix()
	epochTimeString := strconv.FormatInt(epochTime, 10)

	// Set Resource id
	d.SetId(fmt.Sprintf("config-%s", epochTimeString))

	return resourceHerokuConfigRead(d, m)
}

func resourceHerokuConfigRead(d *schema.ResourceData, m interface{}) (err error) {
	err = d.Set("vars", d.Get("vars").(map[string]interface{}))
	err = d.Set("sensitive_vars", d.Get("sensitive_vars").(map[string]interface{}))

	if err != nil {
		return err
	}

	return nil
}

func resourceHerokuConfigUpdate(d *schema.ResourceData, m interface{}) error {
	var vars, sensitiveVars map[string]interface{}

	if d.HasChange("vars") {
		v := d.Get("vars")
		vs := v.(map[string]interface{})
		log.Printf("[DEBUG] vars: %v", vs)
		vars = vs
	}

	if d.HasChange("sensitive_vars") {
		v := d.Get("sensitive_vars")
		vs := v.(map[string]interface{})
		log.Printf("[DEBUG] sensitive vars: %v", vs)
		sensitiveVars = vs
	}

	// Check for duplicate values. If there are duplicates, error out
	dupeErr := duplicateVarsChecker(vars, sensitiveVars)
	if dupeErr != nil {
		return dupeErr
	}

	// If no duplicates, simply set new values in state.
	return resourceHerokuConfigRead(d, m)
}

func resourceHerokuConfigDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] There is no DELETE for config resource since no data is stored in Heroku. " +
		"Resource will be removed from state.")

	d.SetId("")

	return nil
}

// duplicateVarsChecker looks for duplicate vars and returns an error if any duplicates are found.
func duplicateVarsChecker(vars, sensitiveVars map[string]interface{}) error {
	var dupes []interface{}

	for k := range sensitiveVars {
		if _, ok := vars[k]; ok {
			dupes = append(dupes, k)
		}
	}

	log.Printf("[INFO] List of Duplicate config vars (if any) %s", dupes)

	if len(dupes) > 0 {
		return fmt.Errorf("[ERROR] Detected duplicate config vars: %s", dupes)
	}

	return nil
}

// constructVars takes a map of old vars and new vars and outputs a map[string]*string needed for the API call.
func constructVars(oldVars, newVars map[string]interface{}) map[string]*string {
	vars := make(map[string]*string)

	for k, v := range oldVars {
		if v != nil {
			vars[k] = nil
		}
	}

	for k, v := range newVars {
		if v != nil {
			val := v.(string)
			vars[k] = &val
		}
	}

	return vars
}

// mergeVars combines both non-sensitive and sensitive vars together.
func mergeVars(vars, sensitiveVars map[string]interface{}) map[string]interface{} {
	combined := make(map[string]interface{})

	for k, v := range vars {
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

// getVarDiff returns the old and new variables.
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

// vetVarsForState compares the schema vars against the remote vars and returns two maps of vars to be set in state.
//
// This is used to only set the vars/sensitive vars that were defined in the resource schema as config var sensitivity
// is not a feature native to the Heroku Platform API.
func vetVarsForState(vars, sensitiveVars map[string]interface{}, remoteVars map[string]string) (map[string]string, map[string]string) {
	vettedVars := make(map[string]string)
	vettedSensitiveVars := make(map[string]string)

	// Vet vars and sensitiveVars by checking each key/value pair against what is set in remoteVars
	for k := range vars {
		if value, ok := remoteVars[k]; ok {
			vettedVars[k] = value
		}
	}

	for k := range sensitiveVars {
		if value, ok := remoteVars[k]; ok {
			vettedSensitiveVars[k] = value
		}
	}

	return vettedVars, vettedSensitiveVars
}
