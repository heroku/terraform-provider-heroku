package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuAccountFeature() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAccountFeatureUpdate,
		Read:   resourceHerokuAccountFeatureRead,
		Update: resourceHerokuAccountFeatureUpdate,
		Delete: resourceHerokuAccountFeatureDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAccountFeatureImport,
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

func resourceHerokuAccountFeatureImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	_, accountFeatureName, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}
	d.SetId(d.Id())
	d.Set("name", accountFeatureName)

	readErr := resourceHerokuAccountFeatureRead(d, meta)
	if readErr != nil {
		return nil, readErr
	}

	return []*schema.ResourceData{d}, nil
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

	// Get Account email. We will use a combo of account email + feature UUID as the resource id
	account, err := getAccount(meta)
	if err != nil {
		return err
	}
	accountEmail := account.Email

	d.SetId(buildCompositeID(accountEmail, accountFeature.Name))

	return resourceHerokuAccountFeatureRead(d, meta)
}

func resourceHerokuAccountFeatureRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	featureName := getAccountFeatureName(d)

	accountFeature, err := client.AccountFeatureInfo(context.TODO(), featureName)
	if err != nil {
		return err
	}

	d.Set("name", accountFeature.Name)
	d.Set("description", accountFeature.Description)
	d.Set("state", accountFeature.State)
	d.Set("enabled", accountFeature.Enabled)

	return nil
}

// There is no account feature DELETE endpoint. Behavior will be to set feature to enabled = false
// and remove resource from state.
func resourceHerokuAccountFeatureDelete(d *schema.ResourceData, meta interface{}) error {
	_, err := updateAccountFeature(false, d, meta)
	if err != nil {
		return err
	}

	return nil
}

// utility method to update heroku account feature
func updateAccountFeature(enabled bool, d *schema.ResourceData, meta interface{}) (*heroku.AccountFeature, error) {
	client := meta.(*Config).Api

	featureName := getAccountFeatureName(d)
	opts := heroku.AccountFeatureUpdateOpts{
		Enabled: enabled,
	}

	log.Printf("[DEBUG] Updating Heroku Account Feature...")
	accountFeature, err := client.AccountFeatureUpdate(context.TODO(), featureName, opts)
	if err != nil {
		return nil, fmt.Errorf("Error enabling/disabling feature: %s opts %+v", err, opts)
	}

	return accountFeature, nil
}

func getAccount(meta interface{}) (account *heroku.Account, err error) {
	client := meta.(*Config).Api

	account, err = client.AccountInfo(context.TODO())
	if err != nil {
		return nil, err
	}

	return account, nil
}

func getAccountFeatureName(d *schema.ResourceData) (name string) {
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}

	return name
}
