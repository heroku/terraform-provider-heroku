package heroku

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestHelper_MigrateAppToAppID(t *testing.T) {
	p := Provider()
	d := schema.TestResourceDataRaw(t, p.Schema, nil)

	meta, err := providerConfigure(d)
	if err != nil {
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

	client := meta.(*Config).Api
	client.URL = srv.URL

	existing := terraform.InstanceState{
		ID: "2d0b93be-4e89-4652-b57a-bebf43c1f494",
		Attributes: map[string]string{
			"app": "test-app",
		},
	}
	actual, err := migrateAppToAppID(&existing, client)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if actual.Attributes["app_id"] != expectedID {
		t.Fatalf("expected new 'app_id' attribute: %s, got: %s", expectedID, actual.Attributes["app_id"])
	}
}

func TestHelper_UpgradeAppToAppID(t *testing.T) {
	p := Provider()
	d := schema.TestResourceDataRaw(t, p.Schema, nil)

	meta, err := providerConfigure(d)
	if err != nil {
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

	client := meta.(*Config).Api
	client.URL = srv.URL

	existing := map[string]interface{}{
		"app": "test-app",
	}
	actual, err := upgradeAppToAppID(context.Background(), existing, meta)
	if err != nil {
		t.Fatalf("error migrating state: %s", err)
	}

	if actual["app_id"] != expectedID {
		t.Fatalf("expected new 'app_id' attribute: %s, got: %s", expectedID, actual["app_id"])
	}
}
