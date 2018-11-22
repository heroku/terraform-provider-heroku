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

func resourceHerokuAppConfigVar() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAppConfigVarCreate, // There is no CREATE endpoint for config-vars
		Read:   resourceHerokuAppConfigVarRead,
		Update: resourceHerokuAppConfigVarUpdate,
		Delete: resourceHerokuAppConfigVarDelete,
		// TODO: should we handle scenario where a private var is in the public one?

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},

			"private": {
				Type:      schema.TypeList,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Schema{
					Type:      schema.TypeMap,
					Sensitive: true,
				},
			},

			"all_config_vars": {
				Type:      schema.TypeMap,
				Sensitive: true,
				Computed:  true,
			},
		},
	}
}

func resourceHerokuAppConfigVarCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Create a map of both public/private vars
	var publicVars, privateVars map[string]*string
	if v, ok := d.GetOk("public"); ok {
		publicVars = getConfigVarsDiff(nil, v.([]interface{}))
		log.Printf("[INFO] List of publicVars: *%#v", publicVars)
	}
	if v, ok := d.GetOk("private"); ok {
		privateVars = getConfigVarsDiff(nil, v.([]interface{}))
		log.Printf("[INFO] List of privateVars: *%#v", privateVars)
	}

	// TODO: go through both vars and check to make sure that there aren't any overlapping ones

	// Update Vars
	updateVars(d, client, publicVars, privateVars)

	return resourceHerokuAppConfigVarRead(d, meta)
}

func resourceHerokuAppConfigVarRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Get App Name
	appName := getAppName(d)

	//// Get the App Id that we will use as this resource's Id
	//appUuid := getAppUuid(appName, client)
	configVars, err := client.ConfigVarInfoForApp(context.TODO(), appName)
	if err != nil {
		return err
	}

	d.SetId(appName) // TODO: is just using the appName too generic?

	// Iterate through each public/private vars and get the updated value from remote
	publicVars := map[string]*string{}
	if v, ok := d.GetOk("public"); ok {
		for _, vs := range v.([]interface{}) {
			n, ok := vs.(map[string]interface{})
			if !ok {
				continue
			}

			for k := range n {
				publicVars[k] = configVars[k]
			}
		}

		d.Set("public", publicVars)
	}

	privateVars := map[string]*string{}
	if v, ok := d.GetOk("private"); ok {
		for _, vs := range v.([]interface{}) {
			n, ok := vs.(map[string]interface{})
			if !ok {
				continue
			}

			for k := range n {
				privateVars[k] = configVars[k]
			}
		}

		d.Set("private", privateVars)
	}

	// Set all_config_vars in state
	nonNullVars := map[string]string{}
	for k, v := range configVars {
		if v != nil {
			nonNullVars[k] = *v
		}
	}
	if err := d.Set("all_config_vars", nonNullVars); err != nil {
		log.Printf("[WARN] Error setting all_config_vars: %s", err)
	}

	return nil
}

func resourceHerokuAppConfigVarUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// If the config vars changed, then recalculate those
	var newPublicVars, newPrivateVars map[string]*string
	if d.HasChange("public") {
		o, n := d.GetChange("public")
		if o == nil {
			o = []interface{}{}
		}
		if n == nil {
			n = []interface{}{}
		}

		newPublicVars = getConfigVarsDiff(o.([]interface{}), n.([]interface{}))
	}

	if d.HasChange("private") {
		o, n := d.GetChange("private")
		if o == nil {
			o = []interface{}{}
		}
		if n == nil {
			n = []interface{}{}
		}

		newPrivateVars = getConfigVarsDiff(o.([]interface{}), n.([]interface{}))
	}

	// Merge the vars
	updateVars(d, client, newPublicVars, newPrivateVars)

	return resourceHerokuAppConfigVarRead(d, meta)
}

// Removing the app_config_var resource means moving all the config vars defined in this resource
func resourceHerokuAppConfigVarDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting public & private vars")

	// Set all defined variables in the schema to null and update

	publicVars := map[string]*string{}
	if v, ok := d.GetOk("public"); ok {
		for _, vs := range v.([]interface{}) {
			n, ok := vs.(map[string]interface{})
			if !ok {
				continue
			}

			for k := range n {
				publicVars[k] = nil
			}
		}
	}

	privateVars := map[string]*string{}
	if v, ok := d.GetOk("private"); ok {
		for _, vs := range v.([]interface{}) {
			n, ok := vs.(map[string]interface{})
			if !ok {
				continue
			}

			for k := range n {
				privateVars[k] = nil
			}
		}
	}

	updateVars(d, client, publicVars, privateVars)

	d.SetId("")

	return nil
}

func updateVars(d *schema.ResourceData, client *heroku.Service, public, private map[string]*string) error {
	// Get App Name
	appName := getAppName(d)

	// Add privateVars to publicVars as Heroku API does not have 'types' of config vars
	configVars := mergeMaps(public, private)

	log.Printf("[INFO] After merging publicVars & privateVars, here are all the config vars: *%#v", configVars)

	log.Printf("[INFO] Updating %s's config vars: *%#v", appName, configVars) //TODO: need to output the actual value not address
	if _, err := client.ConfigVarUpdate(context.TODO(), appName, configVars); err != nil {
		return fmt.Errorf("[ERROR] Error updating %s's config vars: %s", appName, err)
	}

	// Wait for new release
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
		Refresh: releaseStateRefreshFunc(client, appName, releases[0].ID),
		Timeout: 20 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for new release (%s) to succeed: %s", releases[0].ID, err)
	}

	return nil
}

func mergeMaps(maps ...map[string]*string) map[string]*string {
	result := make(map[string]*string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func getConfigVarsDiff(old []interface{}, new []interface{}) (vars map[string]*string) {
	vars = make(map[string]*string)

	for _, v := range old {
		if v != nil {
			for k := range v.(map[string]interface{}) {
				vars[k] = nil
			}
		}
	}
	for _, v := range new {
		if v != nil {
			for k, v := range v.(map[string]interface{}) {
				val := v.(string)
				vars[k] = &val
			}
		}
	}

	log.Printf("[INFO] Config vars difference: *%#v", vars)

	return vars
}
