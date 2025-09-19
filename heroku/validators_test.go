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

func TestValidateArtifactForGeneration(t *testing.T) {
	// Test Cedar generation
	t.Run("Cedar generation", func(t *testing.T) {
		// Valid: Cedar + slug_id
		err := validateArtifactForGeneration("cedar", true, false)
		if err != nil {
			t.Fatalf("Cedar + slug_id should be valid: %v", err)
		}

		// Invalid: Cedar + oci_image
		err = validateArtifactForGeneration("cedar", false, true)
		if err == nil {
			t.Fatal("Cedar + oci_image should be invalid")
		}
		expectedMsg := "cedar generation apps must use slug_id, not oci_image"
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}

		// Invalid: Cedar + no slug_id
		err = validateArtifactForGeneration("cedar", false, false)
		if err == nil {
			t.Fatal("Cedar without slug_id should be invalid")
		}
		expectedMsg = "cedar generation apps require slug_id"
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	// Test Fir generation
	t.Run("Fir generation", func(t *testing.T) {
		// Valid: Fir + oci_image
		err := validateArtifactForGeneration("fir", false, true)
		if err != nil {
			t.Fatalf("Fir + oci_image should be valid: %v", err)
		}

		// Invalid: Fir + slug_id
		err = validateArtifactForGeneration("fir", true, false)
		if err == nil {
			t.Fatal("Fir + slug_id should be invalid")
		}
		expectedMsg := "fir generation apps must use oci_image, not slug_id"
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}

		// Invalid: Fir + no oci_image
		err = validateArtifactForGeneration("fir", false, false)
		if err == nil {
			t.Fatal("Fir without oci_image should be invalid")
		}
		expectedMsg = "fir generation apps require oci_image"
		if err.Error() != expectedMsg {
			t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	// Test unknown generation (should pass through)
	t.Run("Unknown generation", func(t *testing.T) {
		err := validateArtifactForGeneration("unknown", true, false)
		if err != nil {
			t.Fatalf("Unknown generation should pass through: %v", err)
		}

		err = validateArtifactForGeneration("unknown", false, true)
		if err != nil {
			t.Fatalf("Unknown generation should pass through: %v", err)
		}
	})
}
