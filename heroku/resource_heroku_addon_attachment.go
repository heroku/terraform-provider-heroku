package heroku

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
)

func resourceHerokuAddonAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAddonAttachmentCreate,
		Read:   resourceHerokuAddonAttachmentRead,
		Delete: resourceHerokuAddonAttachmentDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,
		MigrateState:  resourceHerokuAddonAttachmentMigrateState,

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"addon_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceHerokuAddonAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.AddOnAttachmentCreateOpts{Addon: d.Get("addon_id").(string), App: d.Get("app_id").(string)}

	if v := d.Get("name").(string); v != "" {
		opts.Name = &v
	}

	log.Printf("[DEBUG] Addon Attachment create configuration: %#v", opts)
	a, err := client.AddOnAttachmentCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(a.ID)
	log.Printf("[INFO] Addon Attachment ID: %s", d.Id())

	return resourceHerokuAddonAttachmentRead(d, meta)
}

func resourceHerokuAddonAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	match, err := regexp.MatchString(`^[0-9a-f]+-[0-9a-f]+-[0-9a-f]+-[0-9a-f]+-[0-9a-f]+$`, d.Id())
	if !match {
		return fmt.Errorf("You can only import addon attachments by their unique ID")
	}

	addonattachment, err := client.AddOnAttachmentInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving addon attachment: %s", err)
	}

	d.Set("app_id", addonattachment.App.Name)
	d.Set("addon_id", addonattachment.Addon.ID)
	d.Set("name", addonattachment.Name)

	return nil
}

func resourceHerokuAddonAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting Addon Attachment: %s", d.Id())

	// Destroy the app
	_, err := client.AddOnAttachmentDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting addon attachment: %s", err)
	}

	d.SetId("")
	return nil
}
