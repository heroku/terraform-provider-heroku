package heroku

import (
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

// validateUUID matches type terraform.SchemaValidateFunc
func validateUUID(val interface{}, key string) ([]string, []error) {
	s, ok := val.(string)
	if !ok {
		return nil, []error{fmt.Errorf("%q is an invalid UUID: unable to assert %q to string", key, val)}
	}
	if _, err := uuid.Parse(s); err != nil {
		return nil, []error{fmt.Errorf("%q is an invalid UUID: %s", key, err)}
	}
	return nil, nil
}

// validateOCIImage validates that a string is either a valid UUID or SHA256 digest
// OCI images can be referenced by UUID (Heroku internal) or SHA256 digest
func validateOCIImage(val interface{}, key string) ([]string, []error) {
	s, ok := val.(string)
	if !ok {
		return nil, []error{fmt.Errorf("%q is an invalid OCI image identifier: unable to assert %q to string", key, val)}
	}

	// Try UUID first (most common for Heroku internal images)
	if _, err := uuid.Parse(s); err == nil {
		return nil, nil
	}

	// Try SHA256 digest format (sha256:hexstring)
	sha256Pattern := regexp.MustCompile(`^sha256:[a-fA-F0-9]{64}$`)
	if sha256Pattern.MatchString(s) {
		return nil, nil
	}

	// Also accept bare SHA256 hex (64 chars)
	bareHexPattern := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	if bareHexPattern.MatchString(s) {
		return nil, nil
	}

	return nil, []error{fmt.Errorf("%q is an invalid OCI image identifier: must be a UUID or SHA256 digest (sha256:hex or bare hex)", key)}
}
