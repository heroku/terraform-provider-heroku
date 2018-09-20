package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

func resourceHerokuAddonAttachmentMigrateState(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	conn := meta.(*Config)

	log.Printf("[DEBUG] Current version of state file is: v%v", v)

	switch v {
	case 0:
		log.Println("[INFO] Found Heroku Addon Attachment state v0; migrating to v1")
		return migrateAddonAttachmentStateV0toV1(is, conn)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

// Migrate attachment addons in state file to use the UUID for addon_id instead of the NAME for addon_id
func migrateAddonAttachmentStateV0toV1(is *terraform.InstanceState, config *Config) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty Heroku Addon Attachment State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Addon Attachment Attributes before migration: %#v", is.Attributes)

	attachmentAppId := is.Attributes["app_id"]
	attachmentAddOnId := is.Attributes["id"]

	addon, err := config.Api.AddOnInfoByApp(context.TODO(), attachmentAppId, attachmentAddOnId)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving addon: %s", err)
	}

	addonId := addon.ID
	if attachmentAddOnId != addonId {
		log.Printf("[DEBUG] Setting attachment's addon_id to %s", addonId)
		is.Attributes["addon_id"] = addonId
	}

	log.Printf("[DEBUG] Addon Attachment Attributes after migration: %#v", is.Attributes)

	return is, nil
}
