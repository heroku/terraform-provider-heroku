package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
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
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},

			"private": {
				Type:      schema.TypeSet,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},
		},
	}
}

func resourceHerokuAppConfigVarCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Get App Name
	appName := getAppName(d)

	// Define the Public & Private vars
	var publicVars, privateVars map[string]interface{}
	if v, ok := d.GetOk("public"); ok {
		publicVars = v.(map[string]interface{})
	}
	if v, ok := d.GetOk("private"); ok {
		privateVars = v.(map[string]interface{})
	}

	// Combine `public` & `private` config vars together and remove duplicates
	configVars := mergeMaps(publicVars, privateVars)

	// Create Config Vars for App
	log.Printf("[INFO] Creating %s's config vars: *%#v", appName, configVars)

	if _, err := client.ConfigVarUpdate(context.TODO(), appName, configVars); err != nil {
		return fmt.Errorf("[ERROR] Error creating %s's config vars: %s", appName, err)
	}

	return resourceHerokuAppConfigVarRead(d, meta)
}

func resourceHerokuAppConfigVarRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	// Get App Name
	appName := getAppName(d)

	//// Get the App Id that we will use as this resource's Id
	//appUuid := getAppUuid(appName, client)
	configVars, err := client.ConfigVarInfoForApp(context.TODO(), appName)
	if err != nil {
		return err
	}

	d.SetId(appName)
	if err := d.Set("config_vars", configVars); err != nil {
		log.Printf("[WARN] Error setting config vars: %s", err)
	}

	return nil
}

func resourceHerokuAppConfigVarUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	// Determine if public vars have changed
	var oldPublicVars, newPublicVars interface{}
	if d.HasChange("public") {
		oldPublicVars, newPublicVars = d.GetChange("public")

		if oldPublicVars == nil {
			oldPublicVars = []interface{}{}
		}
		if newPublicVars == nil {
			newPublicVars = []interface{}{}
		}
	}

	// Determine if private vars have changed
	var oldPrivateVars, newPrivateVars interface{}
	if d.HasChange("public") {
		oldPrivateVars, newPrivateVars = d.GetChange("private")

		if oldPrivateVars == nil {
			oldPrivateVars = []interface{}{}
		}
		if newPrivateVars == nil {
			newPrivateVars = []interface{}{}
		}
	}

	// Merge old and public vars together
	oldVars := []interface{}{}
	o := append(oldVars, oldPrivateVars)
	o = append(oldVars, oldPublicVars)

	newVars := []interface{}{}
	n := append(newVars, newPrivateVars)
	n = append(newVars, newPublicVars)

	// Update Vars
	err := updateConfigVars(
		d.Id(), client, o, n)
	if err != nil {
		return err
	}

	return nil
}

// Removing the app_config_var resource means moving all config vars from the given app
func resourceHerokuAppConfigVarDelete(d *schema.ResourceData, meta interface{}) error {
	// Essentially perform an Update and then remove the resource from State
	resourceHerokuAppConfigVarUpdate(d, meta)

	d.SetId("")
	return nil
}

func mergeMaps(maps ...map[string]interface{}) map[string]*string {
	result := make(map[string]*string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v.(*string)
		}
	}
	return result
}
