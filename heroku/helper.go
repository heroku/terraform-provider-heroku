package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"

	heroku "github.com/heroku/heroku-go/v6"
	"github.com/heroku/terraform-provider-heroku/v5/version"
)

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
