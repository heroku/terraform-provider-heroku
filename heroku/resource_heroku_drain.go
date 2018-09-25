package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
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
			"url": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

	app, id := parseCompositeID(d.Id())

	dr, err := client.LogDrainInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(dr.ID)
	d.Set("app", app)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDrainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)
	url := d.Get("url").(string)

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

	d.SetId(dr.ID)
	d.Set("url", dr.URL)
	d.Set("token", dr.Token)

	log.Printf("[INFO] Drain ID: %s", d.Id())
	return nil
}

func resourceHerokuDrainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting drain: %s", d.Id())

	// Destroy the drain
	_, err := client.LogDrainDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting drain: %s", err)
	}

	return nil
}

func resourceHerokuDrainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	dr, err := client.LogDrainInfo(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving drain: %s", err)
	}

	d.Set("url", dr.URL)
	d.Set("token", dr.Token)

	return nil
}
