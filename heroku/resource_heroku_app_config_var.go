package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
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
	var publicVars, privateVars *schema.Set
	if v, ok := d.GetOk("public"); ok {
		publicVars = v.(*schema.Set)
	}
	if v, ok := d.GetOk("private"); ok {
		privateVars = v.(*schema.Set)
	}

	// Combine `public` & `private` config vars together and remove duplicates
	allConfigVars := publicVars.Union(privateVars)

	// TODO: remove these before going live
	log.Printf("[INFO] this is publicVars: *%#v", publicVars)
	log.Printf("[INFO] this is privateVars: *%#v", privateVars)
	log.Printf("[INFO] this is allConfigVars: *%#v", allConfigVars)

	allConfigVarsMap := make(map[string]*string)
	for _, vars := range allConfigVars.List() {
		for k, v := range vars.(map[string]interface{}) {
			value := v.(string)
			allConfigVarsMap[k] = &value
		}
	}

	log.Printf("[INFO] Creating %s's config vars: *%#v", appName, allConfigVarsMap) //TODO: need to output the actual value not address
	if _, err := client.ConfigVarUpdate(context.TODO(), appName, allConfigVarsMap); err != nil {
		return fmt.Errorf("[ERROR] Error creating %s's config vars: %s", appName, err)
	}

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
	if v, ok := d.GetOk("public"); ok {
		publicVarsSet := v.(*schema.Set)
		for _, valueMap := range publicVarsSet.List() {
			valueMapFormatted := valueMap.(map[string]interface{})
			for k, _ := range valueMapFormatted {
				if _, ok := valueMapFormatted[k]; !ok {
					publicVarsSet.Add(map[string]*string{k: configVars[k]})
				}
			}
		}

		d.Set("public", publicVarsSet)
	}

	if v, ok := d.GetOk("private"); ok {
		privateVarsSet := v.(*schema.Set)
		for _, valueMap := range privateVarsSet.List() {
			valueMapFormatted := valueMap.(map[string]interface{})
			for k, _ := range valueMapFormatted {
				if _, ok := valueMapFormatted[k]; !ok {
					privateVarsSet.Add(map[string]*string{k: configVars[k]})
				}
			}
		}

		d.Set("private", privateVarsSet)
	}

	return nil
}

func resourceHerokuAppConfigVarUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

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

	// Merge old public and private vars together
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
