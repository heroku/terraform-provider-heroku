package heroku

import "testing"

func TestValidateUUID(t *testing.T) {
	valid := []interface{}{
		"4812ccbc-2a2e-4c6c-bae4-a3d04ed51c0e",
	}
	for _, v := range valid {
		_, errors := validateUUID(v, "id")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid UUID: %q", v, errors)
		}
	}

	invalid := []interface{}{
		"foobarbaz",
		"my-app-name",
		1,
	}
	for _, v := range invalid {
		_, errors := validateUUID(v, "id")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid UUID", v)
		}
	}
}
