package heroku

import (
	"context"
	"fmt"

	heroku "github.com/heroku/heroku-go/v6"
)

// validateOwnerSupportsOtel checks if the owner (app or space) supports OpenTelemetry drains.
// Extracted from the original SDKv2 resource_heroku_telemetry_drain.go so it survives that
// resource's migration to the terraform-plugin-framework.
func validateOwnerSupportsOtel(client *heroku.Service, ownerID, ownerType string) error {
	switch ownerType {
	case "app":
		app, err := client.AppInfo(context.TODO(), ownerID)
		if err != nil {
			return fmt.Errorf("error fetching app info: %s", err)
		}

		if !IsFeatureSupported(app.Generation.Name, "app", "otel") {
			return fmt.Errorf("telemetry drains are only supported for Fir generation apps. App '%s' is %s generation. Use heroku_drain for Cedar apps", app.Name, app.Generation.Name)
		}

	case "space":
		space, err := client.SpaceInfo(context.TODO(), ownerID)
		if err != nil {
			return fmt.Errorf("error fetching space info: %s", err)
		}

		if !IsFeatureSupported(space.Generation.Name, "space", "otel") {
			return fmt.Errorf("telemetry drains are only supported for Fir generation spaces. Space '%s' is %s generation", space.Name, space.Generation.Name)
		}

	default:
		return fmt.Errorf("invalid owner_type: %s", ownerType)
	}

	return nil
}
