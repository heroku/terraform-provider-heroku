package heroku

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuAppFeature() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAppFeatureCreate,
		Update: resourceHerokuAppFeatureUpdate,
		Read:   resourceHerokuAppFeatureRead,
		Delete: resourceHerokuAppFeatureDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAppFeatureImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuAppFeatureV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeHerokuAppFeatureV1,
				Version: 0,
			},
		},
	}
}

func resourceHerokuAppFeatureImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	readErr := resourceHerokuAppFeatureRead(d, meta)
	if readErr != nil {
		return nil, readErr
	}

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuAppFeatureRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return err
	}

	feature, err := client.AppFeatureInfo(context.TODO(), app, id)
	if err != nil {
		return err
	}

	d.Set("app_id", app)
	d.Set("name", feature.Name)
	d.Set("enabled", feature.Enabled)

	return nil
}

func resourceHerokuAppFeatureCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app_id").(string)
	featureName := d.Get("name").(string)
	enabled := d.Get("enabled").(bool)

	opts := heroku.AppFeatureUpdateOpts{Enabled: enabled}

	log.Printf("[DEBUG] Feature set configuration: %#v, %#v", featureName, opts)

	feature, err := client.AppFeatureUpdate(context.TODO(), app, featureName, opts)
	if err != nil {
		return err
	}

	d.SetId(buildCompositeID(app, feature.ID))

	return resourceHerokuAppFeatureRead(d, meta)
}

func resourceHerokuAppFeatureUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("enabled") {
		return resourceHerokuAppFeatureCreate(d, meta)
	}

	return resourceHerokuAppFeatureRead(d, meta)
}

func resourceHerokuAppFeatureDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return err
	}
	featureName := d.Get("name").(string)

	log.Printf("[INFO] Deleting app feature %s (%s) for app %s", featureName, id, app)
	opts := heroku.AppFeatureUpdateOpts{Enabled: false}
	_, err = client.AppFeatureUpdate(context.TODO(), app, id, opts)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceHerokuAppFeatureV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func upgradeHerokuAppFeatureV1(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	upgradedState, err := upgradeAppToAppID(ctx, rawState, meta)
	if err != nil {
		return nil, err
	}

	// Replace special composite ID's app name with app ID.
	_, featureID, err := parseCompositeID(upgradedState["id"].(string))
	if err != nil {
		return nil, err
	}
	upgradedState["id"] = buildCompositeID(upgradedState["app_id"].(string), featureID)

	return upgradedState, nil
}
