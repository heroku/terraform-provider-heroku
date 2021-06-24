package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuCert() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "This resource is deprecated in favor of `heroku_ssl`.",

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
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
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
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	ep, err := client.SSLEndpointInfo(context.Background(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(ep.ID)
	setErr := d.Set("app", app)
	if setErr != nil {
		return nil, setErr
	}

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuCertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)
	preprocess := true
	opts := heroku.SSLEndpointCreateOpts{
		CertificateChain: d.Get("certificate_chain").(string),
		Preprocess:       &preprocess,
		PrivateKey:       d.Get("private_key").(string)}

	log.Printf("[DEBUG] SSL Certificate create configuration: %#v, %#v", app, opts)
	a, err := client.SSLEndpointCreate(context.TODO(), app, opts)
	if err != nil {
		return fmt.Errorf("Error creating SSL endpoint: %s", err)
	}

	d.SetId(a.ID)
	log.Printf("[INFO] SSL Certificate ID: %s", d.Id())

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	cert, err := resourceHerokuSSLCertRetrieve(
		d.Get("app").(string), d.Id(), client)
	if err != nil {
		return err
	}

	d.Set("certificate_chain", cert.CertificateChain)
	d.Set("name", cert.Name)
	d.Set("cname", cert.CName)

	return nil
}

func resourceHerokuCertUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)
	preprocess := true
	opts := heroku.SSLEndpointUpdateOpts{
		CertificateChain: heroku.String(d.Get("certificate_chain").(string)),
		Preprocess:       &preprocess,
		PrivateKey:       heroku.String(d.Get("private_key").(string)),
	}

	if d.HasChange("certificate_chain") || d.HasChange("private_key") {
		log.Printf("[DEBUG] SSL Certificate update configuration: %#v, %#v", app, opts)
		_, err := client.SSLEndpointUpdate(context.TODO(), app, d.Id(), opts)
		if err != nil {
			return fmt.Errorf("Error updating SSL endpoint: %s", err)
		}
	}

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting SSL Cert: %s", d.Id())

	// Destroy the app
	_, err := client.SSLEndpointDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSL Cert: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuSSLCertRetrieve(app string, id string, client *heroku.Service) (*heroku.SSLEndpoint, error) {
	addon, err := client.SSLEndpointInfo(context.TODO(), app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving SSL Cert: %s", err)
	}

	return addon, nil
}
