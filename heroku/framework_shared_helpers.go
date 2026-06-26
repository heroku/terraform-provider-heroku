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

// validateArtifactForApp validates that the release artifact type matches the
// app's build system rather than its generation alone. Cloud Native Buildpack
// apps release OCI images; slug-based apps release slugs. Whether an app uses
// CNB depends on both generation and stack (see IsCNBApp): Fir apps always use
// CNB, and Cedar apps use CNB when stack is "cnb". Keying off the stack (not
// just the generation) is what allows Cedar apps on the "cnb" stack — which
// release OCI images — to be validated correctly.
//
// Extracted from resource_heroku_app_release.go; still used by validators_test.go.
func validateArtifactForApp(generation string, stack string, hasSlug bool, hasOci bool) error {
	switch generation {
	case "cedar", "fir":
		if IsCNBApp(generation, stack) {
			// Fir apps, and Cedar apps on the "cnb" stack, release OCI images.
			if hasSlug {
				return fmt.Errorf("cloud native buildpack apps (generation %q, stack %q) must use oci_image, not slug_id", generation, stack)
			}
			if !hasOci {
				return fmt.Errorf("cloud native buildpack apps (generation %q, stack %q) require oci_image", generation, stack)
			}
			return nil
		}
		// Classic Cedar apps (any non-"cnb" stack) release slugs.
		if hasOci {
			return fmt.Errorf("slug-based apps (generation %q, stack %q) must use slug_id, not oci_image", generation, stack)
		}
		if !hasSlug {
			return fmt.Errorf("slug-based apps (generation %q, stack %q) require slug_id", generation, stack)
		}
	default:
		// Unknown generation - let the API handle it
		return nil
	}

	return nil
}
