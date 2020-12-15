package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

// Global lock to prevent parallelism for heroku_addon since
// the Heroku API cannot handle a single application requesting
// multiple addons simultaneously.
var addonLock sync.Mutex

func resourceHerokuAddon() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAddonCreate,
		Read:   resourceHerokuAddonRead,
		Update: resourceHerokuAddonUpdate,
		Delete: resourceHerokuAddonDelete,
		Exists: resourceHerokuAddonExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 2,
		MigrateState:  resourceHerokuAddonMigrate,

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateCustomAddonName,
			},

			"attachment_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"config": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"provider_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"config_vars": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func validateCustomAddonName(v interface{}, k string) (ws []string, errors []error) {
	// Check length
	v1 := validation.StringLenBetween(1, 256)
	_, errs1 := v1(v, k)
	for _, err := range errs1 {
		errors = append(errors, err)
	}

	// Check validity
	valRegex := regexp.MustCompile(`^[a-zA-Z][A-Za-z0-9_-]+$`)
	v2 := validation.StringMatch(valRegex, "Invalid custom addon name: must start with a letter and can only contain lowercase letters, numbers, and dashes")
	_, errs2 := v2(v, k)
	for _, err := range errs2 {
		errors = append(errors, err)
	}

	return ws, errors
}

func resourceHerokuAddonCreate(d *schema.ResourceData, meta interface{}) error {
	addonLock.Lock()
	defer addonLock.Unlock()

	config := meta.(*Config)
	client := config.Api

	app := d.Get("app").(string)
	opts := heroku.AddOnCreateOpts{
		Plan:    d.Get("plan").(string),
		Confirm: &app,
	}

	if c, ok := d.GetOk("config"); ok {
		opts.Config = make(map[string]string)
		for k, v := range c.(map[string]interface{}) {
			opts.Config[k] = v.(string)
		}
	}

	if v := d.Get("name").(string); v != "" {
		opts.Name = &v
	}

	if v := d.Get("attachment_name").(string); v != "" {
		opts.Attachment.Name = &v
	}

	log.Printf("[DEBUG] Addon create configuration: %#v, %#v", app, opts)
	a, err := client.AddOnCreate(context.TODO(), app, opts)
	if err != nil {
		return err
	}

	// Wait for the Addon to be provisioned
	log.Printf("[DEBUG] Waiting for Addon (%s) to be provisioned", a.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"provisioning"},
		Target:  []string{"provisioned"},
		Refresh: AddOnStateRefreshFunc(client, app, a.ID),
		Timeout: time.Duration(config.AddonCreateTimeout) * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Addon (%s) to be provisioned: %s", d.Id(), err)
	}
	log.Printf("[INFO] Addon provisioned: %s", d.Id())

	// This should be only set after the addon provisioning has been fully completed.
	d.SetId(a.ID)
	log.Printf("[INFO] Addon ID: %s", d.Id())

	return resourceHerokuAddonRead(d, meta)
}

func resourceHerokuAddonRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	// Retrieve the addon
	addon, err := resourceHerokuAddonRetrieve(d.Id(), client)
	if err != nil {
		return err
	}

	// Retrieve the addon's attachments to set the addon.attachment_name.
	// By default, the addon will always have at least one initial attachment to the app that the addon is tied to.
	lr := &heroku.ListRange{}
	addonAttachments, attachmentReadErr := client.AddOnAttachmentListByAddOn(context.TODO(), addon.ID, lr)
	if attachmentReadErr != nil {
		return attachmentReadErr
	}

	// Determine the plan. If we were configured without a specific plan,
	// then just avoid the plan altogether (accepting anything that
	// Heroku sends down).
	plan := addon.Plan.Name
	if v := d.Get("plan").(string); v != "" {
		if idx := strings.IndexRune(v, ':'); idx == -1 {
			idx = strings.IndexRune(plan, ':')
			if idx > -1 {
				plan = plan[:idx]
			}
		}
	}

	var setErr error
	setErr = d.Set("name", addon.Name)
	setErr = d.Set("app", addon.App.Name)
	setErr = d.Set("plan", plan)
	setErr = d.Set("provider_id", addon.ProviderID)
	setErr = d.Set("config_vars", addon.ConfigVars)
	setErr = d.Set("attachment_name", addonAttachments[0].ID) // I *think* it's safe to just read the zero index attachment.
	if setErr != nil {
		return setErr
	}

	return nil
}

func resourceHerokuAddonUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	opts := heroku.AddOnUpdateOpts{}

	app := d.Get("app").(string)

	if d.HasChange("plan") {
		opts.Plan = d.Get("plan").(string)
	}

	if d.HasChange("name") {
		n := d.Get("name").(string)
		opts.Name = &n
	}

	ad, updateErr := client.AddOnUpdate(context.TODO(), app, d.Id(), opts)
	if updateErr != nil {
		return updateErr
	}

	// Store the new addon id if applicable
	d.SetId(ad.ID)

	return resourceHerokuAddonRead(d, meta)
}

func resourceHerokuAddonDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting Addon: %s", d.Id())

	// Destroy the app
	_, err := client.AddOnDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting addon: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuAddonExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Config).Api

	_, err := client.AddOnInfo(context.TODO(), d.Id())
	if err != nil {
		if herr, ok := err.(*url.Error).Err.(heroku.Error); ok && herr.ID == "not_found" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceHerokuAddonRetrieve(id string, client *heroku.Service) (*heroku.AddOn, error) {
	addon, err := client.AddOnInfo(context.TODO(), id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving addon: %s", err)
	}

	return addon, nil
}

func resourceHerokuAddonRetrieveByApp(app string, id string, client *heroku.Service) (*heroku.AddOn, error) {
	addon, err := client.AddOnInfoByApp(context.TODO(), app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving addon: %s", err)
	}

	return addon, nil
}

// AddOnStateRefreshFunc returns a resource.StateRefreshFunc that is used to
// watch an AddOn.
func AddOnStateRefreshFunc(client *heroku.Service, appID, addOnID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		addon, err := resourceHerokuAddonRetrieveByApp(appID, addOnID, client)

		if err != nil {
			return nil, "", err
		}

		// The type conversion here can be dropped when the vendored version of
		// heroku-go is updated.
		return (*heroku.AddOn)(addon), addon.State, nil
	}
}
