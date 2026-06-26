package heroku

import (
	"context"
	"fmt"

	uuid "github.com/hashicorp/go-uuid"
)

// resolveAppToAppID mirrors the SDKv2 upgradeAppToAppID state-upgrade helper for
// the terraform-plugin-framework state upgraders. It takes the fuzzy "app"
// value (name or UUID) from a v0 state and any existing "app_id" value, and
// returns the resolved app UUID.
//
// Resolution order matches upgradeAppToAppID exactly:
//  1. If existingAppID is already a valid UUID, return it unchanged.
//  2. If appFuzzyID is a valid UUID, use it as the app_id.
//  3. Otherwise look up the app by name via the API and use its ID.
func resolveAppToAppID(ctx context.Context, config *Config, appFuzzyID, existingAppID string) (string, error) {
	if _, err := uuid.ParseUUID(existingAppID); err == nil {
		return existingAppID, nil
	}

	if _, err := uuid.ParseUUID(appFuzzyID); err == nil {
		return appFuzzyID, nil
	}

	foundApp, err := config.Api.AppInfo(ctx, appFuzzyID)
	if err != nil {
		return "", fmt.Errorf("upgradeAppToAppID error retrieving app '%s': %w", appFuzzyID, err)
	}
	return foundApp.ID, nil
}
