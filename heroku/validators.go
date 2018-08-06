package heroku

import (
	"fmt"

	"github.com/satori/uuid"
)

func validateUUID(v interface{}, k string) (ws []string, errors []error) {
	if _, err := uuid.FromString(v.(string)); err != nil {
		errors = append(errors, fmt.Errorf("%q is an invalid UUID: %s", k, err))
	}
	return
}
