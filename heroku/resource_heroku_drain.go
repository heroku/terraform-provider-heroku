package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				AtLeastOneOf: []string{"url", "sensitive_url"},
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

		if result[3] == "sensitive" {
			isSensitive = true
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

	d.Set("app", app)
	d.Set("token", dr.Token)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDrainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)

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

	log.Printf("[DEBUG] Drain create configuration: %#v, %#v", app, url)

	var dr *heroku.LogDrain
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		d, err := client.LogDrainCreate(context.TODO(), app, heroku.LogDrainCreateOpts{URL: url})
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
	d.Set("app", app)

	return resourceHerokuDrainRead(d, meta)
}

func resourceHerokuDrainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting drain: %s", d.Id())

	// Destroy the drain
	_, err := client.LogDrainDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting drain: %s", err)
	}

	log.Printf("[INFO] Deleted drain: %s", d.Id())

	d.SetId("")

	return nil
}

func resourceHerokuDrainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	dr, err := client.LogDrainInfo(context.TODO(), d.Get("app").(string), d.Id())
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
