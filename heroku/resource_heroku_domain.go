package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
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
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	do, err := client.DomainInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(do.ID)
	d.Set("app", app)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuDomainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)
	hostname := d.Get("hostname").(string)

	log.Printf("[DEBUG] Domain create configuration: %#v, %#v", app, hostname)

	do, err := client.DomainCreate(context.TODO(), app, heroku.DomainCreateOpts{Hostname: hostname})
	if err != nil {
		return err
	}

	d.SetId(do.ID)
	d.Set("hostname", do.Hostname)
	d.Set("cname", do.CName)

	log.Printf("[INFO] Domain ID: %s", d.Id())
	config := meta.(*Config)
	time.Sleep(time.Duration(config.PostDomainCreateDelay) * time.Second)
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

	d.Set("hostname", do.Hostname)
	d.Set("cname", do.CName)

	return nil
}
