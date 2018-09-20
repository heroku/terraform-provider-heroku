package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
)

func resourceHerokuDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuDomainCreate,
		Read:   resourceHerokuDomainRead,
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
		},
	}
}

func resourceHerokuDomainImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)

	app, id := parseCompositeID(d.Id())

	do, err := config.Api.DomainInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(do.ID)
	d.Set("app", app)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDomainCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	app := d.Get("app").(string)
	hostname := d.Get("hostname").(string)

	log.Printf("[DEBUG] Domain create configuration: %#v, %#v", app, hostname)

	do, err := config.Api.DomainCreate(context.TODO(), app, heroku.DomainCreateOpts{Hostname: hostname})
	if err != nil {
		return err
	}

	d.SetId(do.ID)
	d.Set("hostname", do.Hostname)
	d.Set("cname", do.CName)

	log.Printf("[INFO] Domain ID: %s", d.Id())
	return nil
}

func resourceHerokuDomainDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Deleting Domain: %s", d.Id())

	// Destroy the domain
	_, err := config.Api.DomainDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting domain: %s", err)
	}

	return nil
}

func resourceHerokuDomainRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	app := d.Get("app").(string)
	do, err := config.Api.DomainInfo(context.TODO(), app, d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving domain: %s", err)
	}

	d.Set("hostname", do.Hostname)
	d.Set("cname", do.CName)

	return nil
}
