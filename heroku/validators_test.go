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

func TestValidateOCIImage(t *testing.T) {
	valid := []interface{}{
		// Valid UUIDs
		"4812ccbc-2a2e-4c6c-bae4-a3d04ed51c0e",
		"7f668938-7999-48a7-ad28-c24cbd46c51b",
		// Valid SHA256 with prefix
		"sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		// Valid bare SHA256
		"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
	}
	for _, v := range valid {
		_, errors := validateOCIImage(v, "oci_image")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid OCI image identifier: %q", v, errors)
		}
	}

	invalid := []interface{}{
		// Invalid formats
		"foobarbaz",
		"my-app-name",
		"invalid-format-12345",
		// Invalid SHA256 (wrong length)
		"sha256:abcd1234",
		"abcd1234",
		// Invalid SHA256 (wrong prefix)
		"md5:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		// Non-string types
		1,
		true,
		nil,
	}
	for _, v := range invalid {
		_, errors := validateOCIImage(v, "oci_image")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid OCI image identifier", v)
		}
	}
}

func TestValidateArtifactForApp(t *testing.T) {
	// Classic Cedar apps (any non-"cnb" stack) release slugs.
	t.Run("Cedar slug-based stack", func(t *testing.T) {
		// Valid: Cedar + slug_id
		err := validateArtifactForApp("cedar", "heroku-24", true, false)
		if err != nil {
			t.Fatalf("Cedar + slug_id should be valid: %v", err)
		}

		// Invalid: Cedar + oci_image
		err = validateArtifactForApp("cedar", "heroku-24", false, true)
		if err == nil {
			t.Fatal("Cedar + oci_image should be invalid")
		}
		expectedMsg := `slug-based apps (generation "cedar", stack "heroku-24") must use slug_id, not oci_image`
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}

		// Invalid: Cedar + no slug_id
		err = validateArtifactForApp("cedar", "heroku-24", false, false)
		if err == nil {
			t.Fatal("Cedar without slug_id should be invalid")
		}
		expectedMsg = `slug-based apps (generation "cedar", stack "heroku-24") require slug_id`
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	// Cedar apps on the "cnb" stack release OCI images, just like Fir apps.
	t.Run("Cedar cnb stack", func(t *testing.T) {
		// Valid: Cedar + cnb + oci_image
		err := validateArtifactForApp("cedar", "cnb", false, true)
		if err != nil {
			t.Fatalf("Cedar + cnb + oci_image should be valid: %v", err)
		}

		// Invalid: Cedar + cnb + slug_id
		err = validateArtifactForApp("cedar", "cnb", true, false)
		if err == nil {
			t.Fatal("Cedar + cnb + slug_id should be invalid")
		}
		expectedMsg := `cloud native buildpack apps (generation "cedar", stack "cnb") must use oci_image, not slug_id`
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}

		// Invalid: Cedar + cnb + no oci_image
		err = validateArtifactForApp("cedar", "cnb", false, false)
		if err == nil {
			t.Fatal("Cedar + cnb without oci_image should be invalid")
		}
		expectedMsg = `cloud native buildpack apps (generation "cedar", stack "cnb") require oci_image`
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	// Fir apps always release OCI images, regardless of stack.
	t.Run("Fir generation", func(t *testing.T) {
		// Valid: Fir + oci_image
		err := validateArtifactForApp("fir", "fir", false, true)
		if err != nil {
			t.Fatalf("Fir + oci_image should be valid: %v", err)
		}

		// Invalid: Fir + slug_id
		err = validateArtifactForApp("fir", "fir", true, false)
		if err == nil {
			t.Fatal("Fir + slug_id should be invalid")
		}
		expectedMsg := `cloud native buildpack apps (generation "fir", stack "fir") must use oci_image, not slug_id`
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}

		// Invalid: Fir + no oci_image
		err = validateArtifactForApp("fir", "fir", false, false)
		if err == nil {
			t.Fatal("Fir without oci_image should be invalid")
		}
		expectedMsg = `cloud native buildpack apps (generation "fir", stack "fir") require oci_image`
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	// Test unknown generation (should pass through)
	t.Run("Unknown generation", func(t *testing.T) {
		err := validateArtifactForApp("unknown", "heroku-24", true, false)
		if err != nil {
			t.Fatalf("Unknown generation should pass through: %v", err)
		}

		err = validateArtifactForApp("unknown", "heroku-24", false, true)
		if err != nil {
			t.Fatalf("Unknown generation should pass through: %v", err)
		}
	})
}
