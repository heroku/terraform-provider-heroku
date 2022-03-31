package heroku

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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

		SchemaVersion: 3,
		MigrateState:  resourceHerokuAddonMigrate,

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
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

			"config_var_values": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
				Sensitive: true,
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

	plan := d.Get("plan").(string)
	appID := d.Get("app_id").(string)
	app, err := client.AppInfo(context.TODO(), appID)
	if err != nil {
		return fmt.Errorf("Error reading app for addon: %w", err)
	}

	opts := heroku.AddOnCreateOpts{
		Plan:    plan,
		Confirm: &app.Name,
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

	log.Printf("[DEBUG] Addon create configuration: %#v, %#v", appID, opts)
	addon, err := client.AddOnCreate(context.TODO(), appID, opts)
	if err != nil {
		return fmt.Errorf("Error creating addon %s for app %s (%s): %w", plan, app.Name, appID, err)
	}

	// Wait for the Addon to be provisioned
	log.Printf("[DEBUG] Waiting for Addon (%s) to be provisioned", addon.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"provisioning"},
		Target:  []string{"provisioned"},
		Refresh: AddOnStateRefreshFunc(client, appID, addon.ID),
		Timeout: time.Duration(config.AddonCreateTimeout) * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Addon (%s) to be provisioned: %s", addon.ID, err)
	}
	log.Printf("[INFO] Addon provisioned: %s", addon.ID)

	// This should be only set after the addon provisioning has been fully completed.
	d.SetId(addon.ID)
	log.Printf("[INFO] Addon ID: %s", d.Id())

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		configVarValues, err := retrieveSpecificConfigVars(client, addon.App.ID, addon.ConfigVars)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if len(configVarValues) != len(addon.ConfigVars) {
			return resource.RetryableError(fmt.Errorf("Got %d add-on config vars from the app, but expected %d", len(configVarValues), len(addon.ConfigVars)))
		}
		log.Printf("[INFO] Addon config vars are set: %v", addon.ConfigVars)
		return nil
	})
	if err != nil {
		return err
	}

	return resourceHerokuAddonRead(d, meta)
}

func resourceHerokuAddonRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	addon, err := resourceHerokuAddonRetrieve(d.Id(), client)
	if err != nil {
		return err
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

	d.Set("name", addon.Name)
	d.Set("app_id", addon.App.ID)
	d.Set("plan", plan)
	d.Set("provider_id", addon.ProviderID)
	if err := d.Set("config_vars", addon.ConfigVars); err != nil {
		return err
	}

	configVarValues, err := retrieveSpecificConfigVars(client, addon.App.ID, addon.ConfigVars)
	if err != nil {
		return err
	}
	err = d.Set("config_var_values", configVarValues)
	if err != nil {
		return err
	}

	return nil
}

func resourceHerokuAddonUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	opts := heroku.AddOnUpdateOpts{}

	app := d.Get("app_id").(string)

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
	_, err := client.AddOnDelete(context.TODO(), d.Get("app_id").(string), d.Id())
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

func retrieveSpecificConfigVars(client *heroku.Service, appID string, varNames []string) (map[string]string, error) {
	vars, err := client.ConfigVarInfoForApp(context.TODO(), appID)
	if err != nil {
		return nil, err
	}

	nonNullVars := map[string]string{}
	for k, v := range vars {
		if SliceContainsString(varNames, k) && v != nil {
			nonNullVars[k] = *v
		}
	}

	return nonNullVars, nil
}
