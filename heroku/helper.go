package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
