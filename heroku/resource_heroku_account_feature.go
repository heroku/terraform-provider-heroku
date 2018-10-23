package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
)

func resourceHerokuAccountFeature() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAccountFeatureUpdate,
		Read:   resourceHerokuAccountFeatureRead,
		Update: resourceHerokuAccountFeatureUpdate,
		Delete: resourceHerokuAccountFeatureDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// Account Feature endpoint has no CREATE endpoint
// so UPDATE will serve both create/update functionality for this resource.
func resourceHerokuAccountFeatureUpdate(d *schema.ResourceData, meta interface{}) error {
	var enabled bool
	if v, ok := d.GetOk("enabled"); ok {
		enabled = v.(bool)
	}

	accountFeature, err := updateAccountFeature(enabled, d, meta)
	if err != nil {
		return err
	}

	d.SetId(accountFeature.ID)

	return resourceHerokuAccountFeatureRead(d, meta)
}

func resourceHerokuAccountFeatureRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	featureName := getAccountFeatureName(d)

	accountFeature, err := client.AccountFeatureInfo(context.TODO(), featureName)
	if err != nil {
		return err
	}

	d.Set("description", accountFeature.Description)
	d.Set("state", accountFeature.State)

	return nil
}

// There is no account feature DELETE endpoint. Behavior will be to set feature to enabled = false
// and remove resource from state.
func resourceHerokuAccountFeatureDelete(d *schema.ResourceData, meta interface{}) error {
	_, err := updateAccountFeature(false, d, meta)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// utility method to update heroku account feature
func updateAccountFeature(enabled bool, d *schema.ResourceData, meta interface{}) (*heroku.AccountFeature, error) {
	client := meta.(*Config).Api

	featureName := getAccountFeatureName(d)
	opts := heroku.AccountFeatureUpdateOpts{}
	opts.Enabled = enabled

	log.Printf("[DEBUG] Updating Heroku Account Feature...")
	accountFeature, err := client.AccountFeatureUpdate(context.TODO(), featureName, opts)
	if err != nil {
		return nil, fmt.Errorf("Error enabling/disabling feature: %s opts %+v", err, opts)
	}

	return accountFeature, nil
}

func getAccountFeatureName(d *schema.ResourceData) (name string) {
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}

	return name
}
