package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

func resourceHerokuAddonMigrate(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	conn := meta.(*Config)

	log.Printf("[DEBUG] Current version of state file is: v%v", v)

	switch v {
	case 0:
		log.Println("[INFO] Found Heroku Addon state v0; migrating to v1")
		return migrateAddonIdsStateV0toV1(is, conn)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAddonIdsStateV0toV1(is *terraform.InstanceState, client *Config) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty Heroku Addon State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Addon Id before migration: %#v", is.ID)
	log.Printf("[DEBUG] Addon Attributes before migration: %#v", is.Attributes)

	currentAddonId := is.ID
	addonAppId := is.Attributes["app"]

	addon, err := client.Api.AddOnInfoByApp(context.TODO(), addonAppId, currentAddonId)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving addon: %s", err)
	}

	newAddonId := addon.ID
	if currentAddonId != newAddonId {
		log.Printf("[DEBUG] Setting addon id to %s", newAddonId)
		is.Attributes["id"] = newAddonId
		is.ID = newAddonId
	}

	log.Printf("[DEBUG] Addon Id after migration: %#v", is.ID)
	log.Printf("[DEBUG] Addon Attributes after migration: %#v", is.Attributes)

	return is, nil
}
