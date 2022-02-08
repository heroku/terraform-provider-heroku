package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuDrain() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuDrainCreate,
		Read:   resourceHerokuDrainRead,
		Delete: resourceHerokuDrainDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuDrainImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"url": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"sensitive_url"},
			},

			"sensitive_url": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Sensitive:     true,
				ConflictsWith: []string{"url"},
			},

			"token": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuDrainV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeAppToAppID,
				Version: 0,
			},
		},
	}
}

const retryableError = `App hasn't yet been assigned a log channel. Please try again momentarily.`

func resourceHerokuDrainImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	result := strings.Split(d.Id(), ":")

	var app, id string
	var isSensitive bool
	switch len(result) {
	case 2:
		app = result[0]
		id = result[1]
		isSensitive = false
	case 3:
		app = result[0]
		id = result[1]

		if result[2] == "sensitive" {
			isSensitive = true
		} else {
			return nil, fmt.Errorf("to import a heroku_drain with a sensitive url, please use 'sensitive', not '%s'",
				result[2])
		}
	default:
		return nil, fmt.Errorf("the heroku_drain import ID should consist of 2 or 3 strings separated by a colon")
	}

	dr, err := client.LogDrainInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(dr.ID)

	if isSensitive {
		d.Set("sensitive_url", dr.URL)
	} else {
		d.Set("url", dr.URL)
	}

	foundApp, err := resourceHerokuAppRetrieve(app, client)
	if err != nil {
		return nil, err
	}

	d.Set("app_id", foundApp.App.ID)
	d.Set("token", dr.Token)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDrainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appID := d.Get("app_id").(string)

	var url string
	if v, ok := d.GetOk("url"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] drain url: %s", vs)
		url = vs
	}

	if v, ok := d.GetOk("sensitive_url"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] drain sensitive_url: %s", vs)
		url = vs
	}

	log.Printf("[DEBUG] Drain create configuration: %#v, %#v", appID, url)

	var dr *heroku.LogDrain
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		d, err := client.LogDrainCreate(context.TODO(), appID, heroku.LogDrainCreateOpts{URL: url})
		if err != nil {
			if strings.Contains(err.Error(), retryableError) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		dr = d
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Drain ID: %s", d.Id())

	d.SetId(dr.ID)

	return resourceHerokuDrainRead(d, meta)
}

func resourceHerokuDrainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting drain: %s", d.Id())

	// Destroy the drain
	_, err := client.LogDrainDelete(context.TODO(), d.Get("app_id").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting drain: %s", err)
	}

	log.Printf("[INFO] Deleted drain: %s", d.Id())

	d.SetId("")

	return nil
}

func resourceHerokuDrainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	dr, err := client.LogDrainInfo(context.TODO(), d.Get("app_id").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving drain: %s", err)
	}

	d.Set("token", dr.Token)

	if _, ok := d.GetOk("url"); ok {
		d.Set("url", dr.URL)
	}

	if _, ok := d.GetOk("sensitive_url"); ok {
		d.Set("sensitive_url", dr.URL)
	}

	return nil
}

func resourceHerokuDrainV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"url": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"sensitive_url"},
			},

			"sensitive_url": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Sensitive:     true,
				ConflictsWith: []string{"url"},
			},

			"token": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
