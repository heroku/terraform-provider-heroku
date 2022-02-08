package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuDomainCreate,
		Read:   resourceHerokuDomainRead,
		Update: resourceHerokuDomainUpdate,
		Delete: resourceHerokuDomainDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuDomainImport,
		},

		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"cname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"sni_endpoint_id": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuDomainV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeAppToAppID,
				Version: 0,
			},
		},
	}
}

func resourceHerokuDomainImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}
	log.Printf("[INFO] Importing Domain: %s on App: %s", id, app)

	do, err := client.DomainInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	populateResource(d, do)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDomainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	appID := d.Get("app_id").(string)
	opts := heroku.DomainCreateOpts{
		Hostname: d.Get("hostname").(string),
	}

	if v := d.Get("sni_endpoint_id").(string); v != "" {
		opts.SniEndpoint = &v
	}

	log.Printf("[DEBUG] Domain create configuration: %#v, %#v", appID, opts)

	do, err := client.DomainCreate(context.TODO(), appID, opts)
	if err != nil {
		return err
	}
	populateResource(d, do)

	config := meta.(*Config)
	time.Sleep(time.Duration(config.PostDomainCreateDelay) * time.Second)

	return nil
}

func resourceHerokuDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	appID := d.Get("app_id").(string)
	opts := heroku.DomainUpdateOpts{}

	if d.HasChange("sni_endpoint_id") {
		v := d.Get("sni_endpoint_id").(string)
		opts.SniEndpoint = &v
	}

	do, err := client.DomainUpdate(context.TODO(), appID, d.Id(), opts)
	if err != nil {
		return err
	}

	populateResource(d, do)

	return nil
}

func resourceHerokuDomainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting Domain: %s", d.Id())

	// Destroy the domain
	_, err := client.DomainDelete(context.TODO(), d.Get("app_id").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting domain: %s", err)
	}

	return nil
}

func resourceHerokuDomainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appID := d.Get("app_id").(string)
	do, err := client.DomainInfo(context.TODO(), appID, d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving domain: %s", err)
	}

	log.Printf("[INFO] Reading Domain: %s", d.Id())
	populateResource(d, do)

	return nil
}

func populateResource(d *schema.ResourceData, do *heroku.Domain) {
	d.SetId(do.ID)
	d.Set("app_id", do.App.ID)
	d.Set("hostname", do.Hostname)
	d.Set("cname", do.CName)
	if v := do.SniEndpoint; v != nil {
		d.Set("sni_endpoint_id", v.ID)
	}
}

func resourceHerokuDomainV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"sni_endpoint_id": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
		},
	}
}
