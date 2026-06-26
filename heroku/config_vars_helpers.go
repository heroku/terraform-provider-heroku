package heroku

import (
	"fmt"
	"log"
)

// This file holds config-var helpers shared across the config-var resources
// (heroku_config, heroku_app_config_association, heroku_pipeline_config_var)
// and heroku_app. They were extracted from the original SDKv2
// resource_heroku_config.go so they survive that resource's migration to the
// terraform-plugin-framework.

// duplicateVarsChecker looks for duplicate vars and returns an error if any duplicates are found.
func duplicateVarsChecker(vars, sensitiveVars map[string]interface{}) error {
	var dupes []interface{}

	for k := range sensitiveVars {
		if _, ok := vars[k]; ok {
			dupes = append(dupes, k)
		}
	}

	log.Printf("[INFO] List of Duplicate config vars (if any) %s", dupes)

	if len(dupes) > 0 {
		return fmt.Errorf("[ERROR] Detected duplicate config vars: %s", dupes)
	}

	return nil
}

// constructVars takes a map of old vars and new vars and outputs a map[string]*string needed for the API call.
func constructVars(oldVars, newVars map[string]interface{}) map[string]*string {
	vars := make(map[string]*string)

	for k, v := range oldVars {
		if v != nil {
			vars[k] = nil
		}
	}

	for k, v := range newVars {
		if v != nil {
			val := v.(string)
			vars[k] = &val
		}
	}

	return vars
}

// mergeVars combines both non-sensitive and sensitive vars together.
func mergeVars(vars, sensitiveVars map[string]interface{}) map[string]interface{} {
	combined := make(map[string]interface{})

	for k, v := range vars {
		if v != nil {
			combined[k] = v
		}
	}

	for k, v := range sensitiveVars {
		if v != nil {
			combined[k] = v
		}
	}

	return combined
}

// vetVarsForState compares the schema vars against the remote vars and returns two maps of vars to be set in state.
//
// This is used to only set the vars/sensitive vars that were defined in the resource schema as config var sensitivity
// is not a feature native to the Heroku Platform API.
func vetVarsForState(vars, sensitiveVars map[string]interface{}, remoteVars map[string]string) (map[string]string, map[string]string) {
	vettedVars := make(map[string]string)
	vettedSensitiveVars := make(map[string]string)

	// Vet vars and sensitiveVars by checking each key/value pair against what is set in remoteVars
	for k := range vars {
		if value, ok := remoteVars[k]; ok {
			vettedVars[k] = value
		}
	}

	for k := range sensitiveVars {
		if value, ok := remoteVars[k]; ok {
			vettedSensitiveVars[k] = value
		}
	}

	return vettedVars, vettedSensitiveVars
}
