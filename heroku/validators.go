package heroku

import (
	"fmt"

	"github.com/google/uuid"
)

// validateUUID matches type terraform.SchemaValidateFunc
func validateUUID(v interface{}, k string) ([]string, []error) {
	s, ok := v.(string)
	if !ok {
		return nil, []error{fmt.Errorf("%q is an invalid UUID: unable to assert %q to string", k, v)}
	}
	if _, err := uuid.Parse(s); err != nil {
		return nil, []error{fmt.Errorf("%q is an invalid UUID: %s", k, err)}
	}
	return nil, nil
}
