package heroku

import (
	"context"
	"fmt"

	heroku "github.com/heroku/heroku-go/v6"
)

// This file preserves package-level helpers that were originally defined in
// SDKv2 resource files which have since been migrated to the
// terraform-plugin-framework and deleted. They are kept here because they are
// still referenced by other (non-deleted) files in the package — framework
// resources, data sources, and tests.

// retryableError is the Heroku API message returned while a newly created app
// has not yet been assigned a log channel. The heroku_drain create flow retries
// on it. Extracted from resource_heroku_drain.go.
const retryableError = `App hasn't yet been assigned a log channel. Please try again momentarily.`

// validateAppSupportsTraditionalDrains checks if the app supports traditional
// log drains (Cedar generation only). Extracted from resource_heroku_drain.go.
func validateAppSupportsTraditionalDrains(client *heroku.Service, appID string) error {
	app, err := client.AppInfo(context.TODO(), appID)
	if err != nil {
		return fmt.Errorf("error fetching app info: %s", err)
	}

	if IsFeatureSupported(app.Generation.Name, "app", "otel") {
		return fmt.Errorf("traditional log drains are not supported for Fir generation apps. App '%s' is %s generation. Use heroku_telemetry_drain for Fir apps", app.Name, app.Generation.Name)
	}

	return nil
}

// resourceHerokuAddonRetrieve fetches an addon by ID. Extracted from
// resource_heroku_addon.go; still used by data_source_heroku_addon.go.
func resourceHerokuAddonRetrieve(id string, client *heroku.Service) (*heroku.AddOn, error) {
	addon, err := client.AddOnInfo(context.TODO(), id)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving addon: %s", err)
	}

	return addon, nil
}

// validateArtifactForGeneration validates that the release artifact type matches
// the app generation. Extracted from resource_heroku_app_release.go; still used
// by validators_test.go.
func validateArtifactForGeneration(generationName string, hasSlug bool, hasOci bool) error {
	switch generationName {
	case "cedar":
		if hasOci {
			return fmt.Errorf("cedar generation apps must use slug_id, not oci_image")
		}
		if !hasSlug {
			return fmt.Errorf("cedar generation apps require slug_id")
		}
	case "fir":
		if hasSlug {
			return fmt.Errorf("fir generation apps must use oci_image, not slug_id")
		}
		if !hasOci {
			return fmt.Errorf("fir generation apps require oci_image")
		}
	default:
		// Unknown generation - let the API handle it
		return nil
	}

	return nil
}
