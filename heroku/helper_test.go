package heroku

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHelper_ResolveAppToAppID(t *testing.T) {
	config := NewConfig()
	if err := config.initializeAPI(); err != nil {
		t.Fatal(err)
	}

	expectedID := "5278d60a-bb29-4f72-8936-41991e01d71e"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, writeErr := w.Write([]byte(`{"id":"` + expectedID + `"}`))
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}))
	defer srv.Close()

	config.Api.URL = srv.URL

	// Fuzzy "app" is a name, no existing app_id: resolves via the API lookup.
	actual, err := resolveAppToAppID(context.Background(), config, "test-app", "")
	if err != nil {
		t.Fatalf("error resolving app to app_id: %s", err)
	}

	if actual != expectedID {
		t.Fatalf("expected resolved app_id: %s, got: %s", expectedID, actual)
	}
}
