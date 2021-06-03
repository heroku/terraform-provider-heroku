package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"sni_endpoint": {
				Type:     schema.TypeString,
				Optional: true,
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

	read(d, do)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDomainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	app := d.Get("app").(string)
	opts := heroku.DomainCreateOpts{
		Hostname: d.Get("hostname").(string),
	}

	if v := d.Get("sni_endpoint").(string); v != "" {
		opts.SniEndpoint = &v
	}

	log.Printf("[DEBUG] Domain create configuration: %#v, %#v", app, opts)

	do, err := client.DomainCreate(context.TODO(), app, opts)
	if err != nil {
		return err
	}
	read(d, do)

	config := meta.(*Config)
	time.Sleep(time.Duration(config.PostDomainCreateDelay) * time.Second)

	return nil
}

func resourceHerokuDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	app := d.Get("app").(string)
	opts := heroku.DomainUpdateOpts{}

	if d.HasChange("sni_endpoint") {
		v := d.Get("sni_endpoint").(string)
		opts.SniEndpoint = &v
	}

	do, err := client.DomainUpdate(context.TODO(), app, d.Id(), opts)
	if err != nil {
		return err
	}

	read(d, do)

	return nil
}

func resourceHerokuDomainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting Domain: %s", d.Id())

	// Destroy the domain
	_, err := client.DomainDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting domain: %s", err)
	}

	return nil
}

func resourceHerokuDomainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)
	do, err := client.DomainInfo(context.TODO(), app, d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving domain: %s", err)
	}

	log.Printf("[INFO] Reading Domain: %s", d.Id())
	read(d, do)

	return nil
}

func read(d *schema.ResourceData, do *heroku.Domain) {
	d.SetId(do.ID)
	d.Set("app", do.App.Name)
	d.Set("hostname", do.Hostname)
	d.Set("cname", do.CName)
	if v := do.SniEndpoint; v != nil {
		d.Set("sni_endpoint", v.ID)
	}
}
