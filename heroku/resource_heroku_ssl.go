package heroku

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuSSL() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "This resource is deprecated in favor of `heroku_ssl`.",

		CreateContext: resourceHerokuSSLCreate,
		ReadContext:   resourceHerokuSSLRead,
		UpdateContext: resourceHerokuSSLUpdate,
		DeleteContext: resourceHerokuSSLDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuSSLImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
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

	app, certID, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	ep, err := client.SniEndpointInfo(context.Background(), app, certID)
	if err != nil {
		return nil, err
	}

	d.SetId(ep.ID)
	d.Set("app_id", ep.App.ID)
	d.Set("certificate_chain", ep.CertificateChain)
	d.Set("name", ep.Name)
	// TODO: need to add d.Set("private_key")

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuSSLCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Config).Api
	appID := getAppId(d)

	opts := heroku.SniEndpointCreateOpts{
		CertificateChain: d.Get("certificate_chain").(string),
		PrivateKey:       d.Get("private_key").(string),
	}

	log.Printf("[DEBUG] Creating SSL certificate for app %#v", appID)

	ep, err := client.SniEndpointCreate(context.TODO(), appID, opts)
	if err != nil {
		return diag.Errorf("Error creating SSL certificate for app %s: %v", appID, err.Error())
	}

	log.Printf("[DEBUG] Created SSL Certificate %s", ep.ID)

	d.SetId(ep.ID)

	return resourceHerokuSSLRead(ctx, d, meta)
}

func resourceHerokuSSLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Config).Api

	ep, err := client.SniEndpointInfo(context.Background(), getAppId(d), d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("app_id", ep.App.ID)
	d.Set("certificate_chain", ep.CertificateChain)
	d.Set("name", ep.Name)
	// TODO: need to add d.Set("private_key")

	return nil
}

func resourceHerokuSSLUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Config).Api

	appID := getAppId(d)

	if d.HasChange("certificate_chain") || d.HasChange("private_key") {
		opts := heroku.SniEndpointUpdateOpts{
			CertificateChain: d.Get("certificate_chain").(string),
			PrivateKey:       d.Get("private_key").(string),
		}

		log.Printf("[DEBUG] Updating SSL Certificate configuration: %#v, %#v", appID, opts)

		_, err := client.SniEndpointUpdate(context.TODO(), appID, d.Id(), opts)
		if err != nil {
			return diag.Errorf("Error updating Sni endpoint: %s", err)
		}

		log.Printf("[DEBUG] Updated SSL Certificate configuration: %#v, %#v", appID, opts)
	}

	return resourceHerokuSSLRead(ctx, d, meta)
}

func resourceHerokuSSLDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting SSL Cert: %s", d.Id())

	_, err := client.SniEndpointDelete(context.TODO(), getAppId(d), d.Id())
	if err != nil {
		return diag.Errorf("Error deleting SSL Cert: %s", err)
	}

	log.Printf("[INFO] Deleted SSL Cert: %s", d.Id())

	d.SetId("")

	return nil
}
