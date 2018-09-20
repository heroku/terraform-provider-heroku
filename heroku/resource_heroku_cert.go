package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
)

func resourceHerokuCert() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuCertCreate,
		Read:   resourceHerokuCertRead,
		Update: resourceHerokuCertUpdate,
		Delete: resourceHerokuCertDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuCertImport,
		},

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"certificate_chain": {
				Type:     schema.TypeString,
				Required: true,
			},

			"private_key": {
				Type:     schema.TypeString,
				Required: true,
			},

			"cname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuCertImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)

	app, id := parseCompositeID(d.Id())

	ep, err := config.Api.SSLEndpointInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(ep.ID)
	d.Set("app", app)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuCertCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	app := d.Get("app").(string)
	preprocess := true
	opts := heroku.SSLEndpointCreateOpts{
		CertificateChain: d.Get("certificate_chain").(string),
		Preprocess:       &preprocess,
		PrivateKey:       d.Get("private_key").(string)}

	log.Printf("[DEBUG] SSL Certificate create configuration: %#v, %#v", app, opts)
	a, err := config.Api.SSLEndpointCreate(context.TODO(), app, opts)
	if err != nil {
		return fmt.Errorf("Error creating SSL endpoint: %s", err)
	}

	d.SetId(a.ID)
	log.Printf("[INFO] SSL Certificate ID: %s", d.Id())

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	cert, err := resourceHerokuSSLCertRetrieve(
		d.Get("app").(string), d.Id(), config)
	if err != nil {
		return err
	}

	d.Set("certificate_chain", cert.CertificateChain)
	d.Set("name", cert.Name)
	d.Set("cname", cert.CName)

	return nil
}

func resourceHerokuCertUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	app := d.Get("app").(string)
	preprocess := true
	rollback := false
	opts := heroku.SSLEndpointUpdateOpts{
		CertificateChain: heroku.String(d.Get("certificate_chain").(string)),
		Preprocess:       &preprocess,
		PrivateKey:       heroku.String(d.Get("private_key").(string)),
		Rollback:         &rollback}

	if d.HasChange("certificate_chain") || d.HasChange("private_key") {
		log.Printf("[DEBUG] SSL Certificate update configuration: %#v, %#v", app, opts)
		_, err := config.Api.SSLEndpointUpdate(context.TODO(), app, d.Id(), opts)
		if err != nil {
			return fmt.Errorf("Error updating SSL endpoint: %s", err)
		}
	}

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Deleting SSL Cert: %s", d.Id())

	// Destroy the app
	_, err := config.Api.SSLEndpointDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSL Cert: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuSSLCertRetrieve(app string, id string, config *Config) (*heroku.SSLEndpoint, error) {
	addon, err := config.Api.SSLEndpointInfo(context.TODO(), app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving SSL Cert: %s", err)
	}

	return addon, nil
}
