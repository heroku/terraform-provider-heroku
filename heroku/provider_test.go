package heroku

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"heroku": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestProviderConfigureUsesHeadersForClient(t *testing.T) {
	p := Provider().(*schema.Provider)
	d := schema.TestResourceDataRaw(t, p.Schema, nil)
	d.Set("headers", `{"X-Custom-Header":"yes"}`)

	client, err := providerConfigure(d)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom-Header"); got != "yes" {
			t.Errorf("got X-Custom-Header: %q, want `yes`", got)
		}

		w.Write([]byte(`{"name":"some-app"}`))
	}))
	defer srv.Close()

	c := client.(*heroku.Service)
	c.URL = srv.URL

	_, err = c.AppInfo(context.Background(), "does-not-matter")
	if err != nil {
		t.Fatal(err)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("HEROKU_API_KEY"); v == "" {
		t.Fatal("HEROKU_API_KEY must be set for acceptance tests")
	}
}

func testAccSkipTestIfOrganizationMissing(t *testing.T) {
	if os.Getenv("HEROKU_ORGANIZATION") == "" {
		t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
	}
}

func testAccSkipTestIfUserMissing(t *testing.T) {
	if os.Getenv("HEROKU_TEST_USER") == "" {
		t.Skip("HEROKU_TEST_USER is not set; skipping test.")
	}
}
