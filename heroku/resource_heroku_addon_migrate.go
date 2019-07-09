package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/terraform"
	heroku "github.com/heroku/heroku-go/v5"
	"log"
	"strconv"
	"strings"
)

func resourceHerokuAddonMigrate(v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	client := meta.(*Config).Api

	log.Printf("[DEBUG] Current version of state file is: v%v", v)

	switch v {
	case 0:
		log.Println("[INFO] Found Heroku Addon state v0; migrating to v1")
		return migrateAddonIdsStateV0toV1(is, client)
	case 1:
		log.Println("[INFO] Found Heroku Addon state v1; migrating to v2")
		return migrateAddonConfigFromListSetToSet(is, client)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateAddonIdsStateV0toV1(is *terraform.InstanceState, client *heroku.Service) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty Heroku Addon State; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Addon Id before migration: %#v", is.ID)
	log.Printf("[DEBUG] Addon Attributes before migration: %#v", is.Attributes)

	currentAddonId := is.ID
	addonAppId := is.Attributes["app"]

	addon, err := client.AddOnInfoByApp(context.TODO(), addonAppId, currentAddonId)
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

func migrateAddonConfigFromListSetToSet(is *terraform.InstanceState, client *heroku.Service) (*terraform.InstanceState, error) {
	if is.Empty() || is.Attributes == nil {
		log.Println("[DEBUG] Empty Heroku Addon State; nothing to migrate.")
		return is, nil
	}

	// Check to see if heroku_addon.config is a TypeList of TypeSet
	log.Printf("Checking if heroku_addon state is the old TypeList of TypeSet")
	if is.Attributes["config.%"] != "" {
		// This means the config attribute is the correct data type of just a TypeSet.
		log.Printf("heroku_addon.config is the correct data type. No migration needed.")
		return is, nil
	}

	// If the execution has gotten this far, this means the config attribute is a TypeList of TypeSets,
	// which means we migrate it to just a TypeSet.
	log.Printf("heroku_addon.config is not the correct data type. Migrating to a TypeList of TypeSet to TypeSet.")

	// Define a map to store the new format of configs.
	configMap := map[string]string{}

	// Get the length & generate a slice
	configLength, convertErr := strconv.Atoi(is.Attributes["config.#"])
	if convertErr != nil {
		return nil, convertErr
	}
	configLengthSlice := makeRange(0, configLength-1)

	for _, i := range configLengthSlice {
		matchStr := fmt.Sprintf("config.%v.", i)
		keys := getAttributeKeys(is.Attributes, matchStr)
		for _, k := range keys {
			oldConfigKey := fmt.Sprintf("config.%v.%s", i, k)
			configMap[k] = is.Attributes[oldConfigKey]

			// Then delete the old key/value pair
			delete(is.Attributes, oldConfigKey)
		}
	}

	// Set the new map of config to its length
	is.Attributes["config.%"] = strconv.Itoa(len(configMap))

	// Set each new config key/value pair.
	for k, v := range configMap {
		is.Attributes[fmt.Sprintf("config.%s", k)] = v
	}

	// Delete the old config TypeList
	delete(is.Attributes, "config.#")

	return is, nil
}

func getAttributeKeys(attrs map[string]string, matchStr string) []string {
	keys := make([]string, 0)
	for k := range attrs {
		if strings.Contains(k, matchStr) {
			kSlice := strings.Split(k, matchStr)
			keys = append(keys, kSlice[1])
		}
	}
	return keys
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}
