package heroku

// Feature matrix system for graceful handling of generation differences
// between Cedar and Fir generations in Terraform Provider Heroku.

// featureMatrix defines which features are supported for each generation and resource type.
// This is the single source of truth for feature availability based on the
// unsupported features data from Platform API's 3.sdk Generation endpoints.
var featureMatrix = map[string]map[string]map[string]bool{
	"cedar": {
		"space": {
			"private":               true, // All spaces are private
			"shield":                true, // Cedar supports shield spaces
			"trusted_ip_ranges":     true,
			"private_vpn":           true,
			"outbound_rules":        true,
			"private_space_logging": true,
			"outbound_ips":          true, // Cedar supports outbound IPs
			"vpn_connection":        true, // Cedar supports VPN connections
			"inbound_ruleset":       true, // Cedar supports inbound rulesets
			"peering_connection":    true, // Cedar supports IPv4 peering
		},
		"build": {
			"traditional_buildpacks":  true,  // Cedar supports buildpack URLs
			"cloud_native_buildpacks": false, // Cedar does not support CNB
		},
	},
	"fir": {
		"space": {
			"private":               true,  // All spaces are private
			"shield":                false, // Fir does not support shield spaces
			"trusted_ip_ranges":     false, // trusted_ip_ranges
			"private_vpn":           false, // private_vpn
			"outbound_rules":        false, // outbound_rules
			"private_space_logging": false, // private_space_logging
			"outbound_ips":          false, // space_outbound_ips
			"vpn_connection":        false, // VPN connections not supported
			"inbound_ruleset":       false, // Inbound rulesets not supported
			"peering_connection":    false, // IPv4 peering not supported
		},
		"build": {
			"traditional_buildpacks":  false, // Fir does not support buildpack URLs
			"cloud_native_buildpacks": true,  // Fir supports CNB via project.toml
		},
	},
}

// IsFeatureSupported checks if a feature is supported for a given generation and resource type.
// Returns true if the feature is supported, false otherwise.
//
// Parameters:
//   - generation: "cedar" or "fir"
//   - resourceType: "space", "app", "build", etc.
//   - feature: "shield", "trusted_ip_ranges", "private_vpn", etc.
//
// Example:
//
//	if IsFeatureSupported("fir", "space", "shield") {
//	    // proceed with shield configuration
//	}
func IsFeatureSupported(generation, resourceType, feature string) bool {
	if gen, exists := featureMatrix[generation]; exists {
		if res, exists := gen[resourceType]; exists {
			if supported, exists := res[feature]; exists {
				return supported
			}
		}
	}

	// Default to false for any unknown generation/resource/feature combination
	return false
}
