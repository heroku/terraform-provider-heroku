package heroku

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
	v5 "github.com/heroku/heroku-go/v5"
)

type Endpoint interface {
	Name() string
	CertificateChain() string
	CName() string
}

type SniEndpoint struct {
	herokuEndpoint *heroku.SniEndpoint
}

type SSLEndpoint struct {
	herokuEndpoint *heroku.SSLEndpoint
}

func (e SniEndpoint) Name() string {
	return e.herokuEndpoint.Name
}

func (e SniEndpoint) CName() string {
	return ""
}

func (e SniEndpoint) CertificateChain() string {
	return e.herokuEndpoint.CertificateChain
}

func (e SSLEndpoint) Name() string {
	return e.herokuEndpoint.Name
}

func (e SSLEndpoint) CName() string {
	return e.herokuEndpoint.CName
}

func (e SSLEndpoint) CertificateChain() string {
	return e.herokuEndpoint.CertificateChain
}

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

			"legacy": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceHerokuLegacyCertImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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
	d.Set("legacy", true)
	setErr := d.Set("app", app)
	if setErr != nil {
		return nil, setErr
	}

	return []*schema.ResourceData{d}, nil
}

func errToHttpError(err error) *v5.Error {
	urlErr, ok := err.(*url.Error)
	if ok {
		v5Err, ok := urlErr.Err.(v5.Error)
		if ok {
			return &v5Err
		}
	}

	return nil
}

func isHttpStatusCode(err error, statusCode int) bool {
	if httpErr := errToHttpError(err); httpErr != nil {
		return httpErr.StatusCode == statusCode
	}

	return false
}

func resourceHerokuCertImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	ep, err := client.SniEndpointInfo(context.Background(), app, id)
	if err != nil {
		if isHttpStatusCode(err, 404) {
			return resourceHerokuLegacyCertImport(d, meta)
		} else {
			return nil, err
		}
	}

	d.SetId(ep.ID)
	setErr := d.Set("app", app)
	if setErr != nil {
		return nil, setErr
	}

	return []*schema.ResourceData{d}, nil
}

func hasLegacySSLEndpoint(client *heroku.Service, appIdentity string) (bool, error) {
	addOns, err := client.AddOnListByApp(context.TODO(), appIdentity, &heroku.ListRange{Field: "id"})
	if err != nil {
		return false, fmt.Errorf("Error looking up add-ons: %s", err)
	}

	for _, a := range addOns {
		if a.Plan.Name == "ssl:endpoint" {
			// TODO list endpoints?
			return true, nil
		}
	}

	return false, nil
}

func resourceHerokuLegacyCertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	app := d.Get("app").(string)
	preprocess := true
	opts := heroku.SSLEndpointCreateOpts{
		CertificateChain: d.Get("certificate_chain").(string),
		Preprocess:       &preprocess,
		PrivateKey:       d.Get("private_key").(string),
	}

	log.Printf("[DEBUG] SSL Certificate create configuration: %#v, %#v", app, opts)
	a, err := client.SSLEndpointCreate(context.TODO(), app, opts)
	if err != nil {
		return fmt.Errorf("Error creating SSL endpoint: %s", err)
	}

	d.SetId(a.ID)
	d.Set("legacy", true)
	log.Printf("[INFO] SSL Certificate ID: %s", d.Id())

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	app := d.Get("app").(string)

	hasLegacySSL, err := hasLegacySSLEndpoint(client, app)
	if err != nil {
		return err
	}
	if hasLegacySSL {
		return resourceHerokuLegacyCertCreate(d, meta)
	}

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

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	legacy := d.Get("legacy").(bool)

	cert, err := resourceHerokuSSLCertRetrieve(legacy, d.Get("app").(string), d.Id(), client)
	if err != nil {
		return err
	}

	d.Set("certificate_chain", cert.CertificateChain())
	d.Set("name", cert.Name())
	d.Set("cname", cert.CName())

	return nil
}

func resourceHerokuCertUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	fmt.Printf("%#v", d.Get("legacy"))
	legacy := d.Get("legacy").(bool)

	app := d.Get("app").(string)

	if d.HasChange("certificate_chain") || d.HasChange("private_key") {
		if legacy {
			preprocess := true
			opts := heroku.SSLEndpointUpdateOpts{
				CertificateChain: heroku.String(d.Get("certificate_chain").(string)),
				Preprocess:       &preprocess,
				PrivateKey:       heroku.String(d.Get("private_key").(string)),
			}

			log.Printf("[DEBUG] SSL Certificate update configuration: %#v, %#v", app, opts)
			_, err := client.SSLEndpointUpdate(context.TODO(), app, d.Id(), opts)
			if err != nil {
				return fmt.Errorf("Error updating SSL endpoint: %s", err)
			}
		} else {
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
	}

	return resourceHerokuCertRead(d, meta)
}

func resourceHerokuCertDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	legacy := d.Get("legacy").(bool)

	log.Printf("[INFO] Deleting SSL Cert: %s", d.Id())

	if legacy {
		_, err := client.SSLEndpointDelete(context.TODO(), d.Get("app").(string), d.Id())
		if err != nil {
			return fmt.Errorf("Error deleting SSL Cert: %s", err)
		}
	} else {
		_, err := client.SniEndpointDelete(context.TODO(), d.Get("app").(string), d.Id())
		if err != nil {
			return fmt.Errorf("Error deleting SSL Cert: %s", err)
		}
	}

	d.SetId("")
	return nil
}

func resourceHerokuSSLCertRetrieve(legacy bool, app string, id string, client *heroku.Service) (Endpoint, error) {
	if legacy {
		endpoint, err := client.SSLEndpointInfo(context.TODO(), app, id)
		if err != nil {
			return nil, fmt.Errorf("Error retrieving SSL Cert: %s", err)
		}

		return &SSLEndpoint{endpoint}, nil
	}
	endpoint, err := client.SniEndpointInfo(context.TODO(), app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving SSL Cert: %s", err)
	}

	return &SniEndpoint{endpoint}, nil
}
