package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
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

func doesHerokuAppExist(appName string, client *Config) (*heroku.App, error) {
	app, err := client.Api.AppInfo(context.TODO(), appName)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[ERROR] Your app does not exist")
	}
	return app, nil
}
