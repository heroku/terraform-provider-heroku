package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuSSL() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSSLCreate,
		Read:   resourceHerokuSSLRead,
		Update: resourceHerokuSSLUpdate,
		Delete: resourceHerokuSSLDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuSSLImport,
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

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuSSLImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	ep, err := client.SniEndpointInfo(context.Background(), app, id)
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

func resourceHerokuSSLCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	app := d.Get("app").(string)

	opts := heroku.SniEndpointCreateOpts{
		CertificateChain: d.Get("certificate_chain").(string),
		PrivateKey:       d.Get("private_key").(string),
	}

	log.Printf("[DEBUG] SSL Certificate create configuration: %#v, %#v", app, opts)
	a, err := client.SniEndpointCreate(context.TODO(), app, opts)
	if err != nil {
		return fmt.Errorf("Error creating SniEndpoint: %s", err)
	}

	d.SetId(a.ID)
	log.Printf("[INFO] SSL Certificate ID: %s", d.Id())

	return resourceHerokuSSLRead(d, meta)
}

func resourceHerokuSSLRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	cert, err := resourceHerokuSSLRetrieve(d.Get("app").(string), d.Id(), client)
	if err != nil {
		return err
	}

	d.Set("certificate_chain", cert.CertificateChain)
	d.Set("name", cert.Name)

	return nil
}

func resourceHerokuSSLUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)

	if d.HasChange("certificate_chain") || d.HasChange("private_key") {
		opts := heroku.SniEndpointUpdateOpts{
			CertificateChain: d.Get("certificate_chain").(string),
			PrivateKey:       d.Get("private_key").(string),
		}

		log.Printf("[DEBUG] SSL Certificate update configuration: %#v, %#v", app, opts)
		_, err := client.SniEndpointUpdate(context.TODO(), app, d.Id(), opts)
		if err != nil {
			return fmt.Errorf("Error updating Sni endpoint: %s", err)
		}
	}

	return resourceHerokuSSLRead(d, meta)
}

func resourceHerokuSSLDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting SSL Cert: %s", d.Id())

	_, err := client.SniEndpointDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSL Cert: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuSSLRetrieve(app string, id string, client *heroku.Service) (*heroku.SniEndpoint, error) {
	endpoint, err := client.SniEndpointInfo(context.TODO(), app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving SSL Cert: %s", err)
	}

	return endpoint, nil
}
