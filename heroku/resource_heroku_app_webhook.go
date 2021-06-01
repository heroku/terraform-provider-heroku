package heroku

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	validation "github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuAppWebhook() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAppWebhookCreate,
		Read:   resourceHerokuAppWebhookRead,
		Update: resourceHerokuAppWebhookUpdate,
		Delete: resourceHerokuAppWebhookDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAppWebhookImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"level": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"notify", "sync"}, true),
			},

			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"include": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"api:addon-attachment",
						"api:addon",
						"api:app",
						"api:build",
						"api:collaborator",
						"api:domain",
						"api:dyno",
						"api:formation",
						"api:release",
						"api:sni-endpoint"}, true),
				},
			},

			"secret": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"authorization": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Callback for schema Resource.Create
func resourceHerokuAppWebhookCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appId := getAppId(d)

	opts := heroku.AppWebhookCreateOpts{
		Level:   d.Get("level").(string),
		URL:     d.Get("url").(string),
		Include: getInclude(d),
	}

	if v, ok := d.GetOk("secret"); ok {
		secret := v.(string)
		opts.Secret = &secret
	}

	if v, ok := d.GetOk("authorization"); ok {
		authorization := v.(string)
		opts.Authorization = &authorization
	}

	webhook, err := client.AppWebhookCreate(context.TODO(), appId, opts)
	if err != nil {
		return err
	}

	d.SetId(webhook.ID)

	return nil
}

// Callback for schema Resource.Read
func resourceHerokuAppWebhookRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	appId := getAppId(d)

	webhook, err := client.AppWebhookInfo(context.TODO(), appId, d.Id())
	if err != nil {
		return err
	}

	d.Set("url", webhook.URL)
	d.Set("level", webhook.Level)
	d.Set("include", webhook.Include)

	return nil
}

// Callback for schema Resource.Update
func resourceHerokuAppWebhookUpdate(d *schema.ResourceData, meta interface{}) error {
	// Enable Partial state mode and what we successfully committed
	d.Partial(true)

	client := meta.(*Config).Api
	opts := heroku.AppWebhookUpdateOpts{}

	appId := getAppId(d)

	if d.HasChange("level") {
		v := d.Get("level").(string)
		log.Printf("[DEBUG] New Level: %s", v)
		opts.Level = &v
	}

	if d.HasChange("url") {
		v := d.Get("url").(string)
		log.Printf("[DEBUG] New URL: %v", v)
		opts.URL = &v
	}

	if d.HasChange("include") {
		v := getIncludeAsPointers(d)
		log.Printf("[DEBUG] New include: %v", v)
		opts.Include = v
	}

	if d.HasChange("secret") {
		if v, ok := d.GetOk("secret"); ok {
			secret := v.(string)
			log.Printf("[DEBUG] New Secret: %s", secret)
			opts.Secret = &secret
		} else {
			log.Printf("[DEBUG] Secret Removed")
			opts.Secret = nil
		}
	}

	if d.HasChange("authorization") {
		if v, ok := d.GetOk("authorization"); ok {
			authorization := v.(string)
			log.Printf("[DEBUG] New Authorization: %s", authorization)
			opts.Authorization = &authorization
		} else {
			log.Printf("[DEBUG] Authorization Removed")
			opts.Authorization = nil
		}
	}

	log.Printf("[DEBUG] Updating Heroku webhook...")
	_, err := client.AppWebhookUpdate(context.TODO(), appId, d.Id(), opts)

	if err != nil {
		return err
	}

	d.Partial(false)

	return nil
}

// Callback for schema Resource.Delete
func resourceHerokuAppWebhookDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	appId := getAppId(d)

	_, err := client.AppWebhookDelete(context.TODO(), appId, d.Id())
	return err
}

func resourceHerokuAppWebhookImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, id, err := parseCompositeID(d.Id())

	webhook, err := client.AppWebhookInfo(context.TODO(), app, id)
	if err != nil {
		return nil, err
	}

	d.SetId(webhook.ID)
	d.Set("app_id", webhook.App.ID)
	d.Set("url", webhook.URL)
	d.Set("level", webhook.Level)
	d.Set("include", webhook.Include)

	return []*schema.ResourceData{d}, nil
}

func getInclude(d *schema.ResourceData) []string {
	rawInclude := d.Get("include").([]interface{})
	include := make([]string, len(rawInclude))

	for i, v := range rawInclude {
		include[i] = v.(string)
	}
	return include
}

func getIncludeAsPointers(d *schema.ResourceData) []*string {
	rawInclude := d.Get("include").([]interface{})
	include := make([]*string, len(rawInclude))

	for i, v := range rawInclude {
		vv := v.(string)
		include[i] = &vv
	}
	return include
}
