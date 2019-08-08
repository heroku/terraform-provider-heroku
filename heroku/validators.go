package heroku

import (
	"fmt"

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
