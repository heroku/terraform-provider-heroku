package heroku

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
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

		//Importer: &schema.ResourceImporter{
		//	State: resourceHerokuConfigImport,
		//},

		Schema: map[string]*schema.Schema{
			"name": {
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

	// Check for duplicate values. If there are duplicates, error out
	dupeErr := duplicateChecker(vars, sensitiveVars)
	if dupeErr != nil {
		return dupeErr
	}

	// Set the ID to be name + epoch time for uniqueness
	name := d.Get("name").(string)
	epochTime := time.Now().Unix()
	epochTimeString := strconv.FormatInt(epochTime, 10)

	// Set Resource id
	d.SetId(fmt.Sprintf("%s-%s", name, epochTimeString))

	return resourceHerokuConfigRead(d, m)
}

func resourceHerokuConfigRead(d *schema.ResourceData, m interface{}) (err error) {
	err = d.Set("name", d.Get("name").(string))
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
	dupeErr := duplicateChecker(vars, sensitiveVars)
	if dupeErr != nil {
		return dupeErr
	}

	return resourceHerokuConfigRead(d, m)
}

func resourceHerokuConfigDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] There is no DELETE for config resource since no data is stored in Heroku. " +
		"Resource will be removed from state.")

	return nil
}

func duplicateChecker(vars, sensitiveVars map[string]interface{}) error {
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
