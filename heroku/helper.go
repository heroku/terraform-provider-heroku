package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	heroku "github.com/heroku/heroku-go/v5"
	"github.com/heroku/terraform-provider-heroku/v4/version"
)

// getAppName extracts the app attribute generically from a Heroku resource.
func getAppName(d *schema.ResourceData) string {
	var appName string
	if v, ok := d.GetOk("app"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App name: %s", vs)
		appName = vs
	}

	return appName
}

// getAppId extracts the app attribute generically from a Heroku resource.
func getAppId(d *schema.ResourceData) string {
	var appName string
	if v, ok := d.GetOk("app_id"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App id name: %s", vs)
		appName = vs
	}

	return appName
}

// getEmail extracts the email attribute generically from a Heroku resource.
func getEmail(d *schema.ResourceData) string {
	var email string
	if v, ok := d.GetOk("email"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] Email: %s", vs)
		email = vs
	}

	return email
}

func getPipelineID(d *schema.ResourceData) string {
	var pipelineID string
	if v, ok := d.GetOk("pipeline_id"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] pipeline ID: %s", vs)
		pipelineID = vs
	}

	return pipelineID
}

func doesHerokuAppExist(appName string, client *heroku.Service) (*heroku.App, error) {
	app, err := client.AppInfo(context.TODO(), appName)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[ERROR] Your app does not exist")
	}
	return app, nil
}

func buildCompositeID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func parseCompositeID(id string) (p1 string, p2 string, err error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		p1 = parts[0]
		p2 = parts[1]
	} else {
		err = fmt.Errorf("error: Import composite ID requires two parts separated by colon, eg x:y")
	}
	return
}

func providerVersion() string {
	return version.ProviderVersion
}

func SliceContainsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// migrateAppToAppID is the older (v0.11) MigrateState helper.
func migrateAppToAppID(is *terraform.InstanceState, client *heroku.Service) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty state; nothing to migrate.")
		return is, nil
	}

	appFuzzyID := is.Attributes["app"]
	appID := is.Attributes["app_id"]

	_, err := uuid.ParseUUID(appID)
	if err == nil {
		// app_id is already a valid UUID
		return is, nil
	}

	_, err = uuid.ParseUUID(appFuzzyID)
	if err == nil {
		is.Attributes["app_id"] = appFuzzyID
	} else {
		foundApp, err := client.AppInfo(context.Background(), appFuzzyID)
		if err != nil {
			return nil, fmt.Errorf("migrateAppToAppID error retrieving app '%s': %w", appFuzzyID, err)
		}
		is.Attributes["app_id"] = foundApp.ID
	}

	return is, nil
}

// upgradeAppToAppID is the newer (v0.12+) StateUpgrade helper.
func upgradeAppToAppID(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	appFuzzyID, _ := rawState["app"].(string)
	appID, _ := rawState["app_id"].(string)

	_, err := uuid.ParseUUID(appID)
	if err == nil {
		// app_id is already a valid UUID
		return rawState, nil
	}

	_, err = uuid.ParseUUID(appFuzzyID)
	if err == nil {
		rawState["app_id"] = appFuzzyID
	} else {
		client := meta.(*Config).Api
		foundApp, err := client.AppInfo(ctx, appFuzzyID)
		if err != nil {
			return nil, fmt.Errorf("upgradeAppToAppID error retrieving app '%s': %w", appFuzzyID, err)
		}
		rawState["app_id"] = foundApp.ID
	}

	return rawState, nil
}
