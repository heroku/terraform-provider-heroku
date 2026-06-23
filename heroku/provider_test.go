package heroku

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	helper "github.com/heroku/terraform-provider-heroku/v5/helper/test"
)

const (
	ProviderNameHeroku = "heroku"
)

// testAccProtoV6ProviderFactories serves the terraform-plugin-framework
// provider (protocol 6) to the acceptance-test harness. It replaces the SDKv2
// testAccProviders/testAccProviderFactories used before the migration.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	ProviderNameHeroku: providerserver.NewProtocol6WithError(NewFrameworkProvider()),
}

// testAccProviderConfig is an initialized *Config built from the environment,
// used by acceptance-test check functions that previously read
// testAccProviderConfig. The framework provider no longer exposes a
// Meta() accessor, so tests use this package-level config instead.
var testAccProviderConfig *Config

var testAccConfig *helper.TestConfig

// newTestAccConfig builds a *Config from environment variables and the netrc
// file, mirroring the framework provider's Configure logic. initializeAPI only
// constructs an HTTP client (no network call), so this is safe to run in init.
func newTestAccConfig() *Config {
	config := NewConfig()

	// Best effort: netrc may not exist in CI.
	_ = config.applyNetrcFile()

	if v := os.Getenv("HEROKU_EMAIL"); v != "" {
		config.Email = v
	}
	if v := os.Getenv("HEROKU_API_KEY"); v != "" {
		config.APIKey = v
	}

	_ = config.initializeAPI()

	return config
}

func init() {
	testAccProviderConfig = newTestAccConfig()
	testAccConfig = helper.NewTestConfig()
}

func TestProvider(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(NewFrameworkProvider())()
	if _, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{}); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderConfigureUsesHeadersForClient(t *testing.T) {
	config := NewConfig()
	config.Headers.Set("X-Custom-Header", "yes")
	if err := config.initializeAPI(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom-Header"); got != "yes" {
			t.Errorf("got X-Custom-Header: %q, want `yes`", got)
		}

		_, writeErr := w.Write([]byte(`{"name":"some-app"}`))
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}))
	defer srv.Close()

	c := config.Api
	c.URL = srv.URL

	if _, err := c.AppInfo(context.Background(), "does-not-matter"); err != nil {
		t.Fatal(err)
	}
}

func testAccPreCheck(t *testing.T) {
	testAccConfig.GetOrAbort(t, helper.TestConfigAPIKey)
}

func createTempConfigFile(content string, name string) (*os.File, error) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), name)
	if err != nil {
		return nil, fmt.Errorf("Error creating temporary test file. err: %s", err.Error())
	}

	_, err = tmpfile.WriteString(content)
	if err != nil {
		removeErr := os.Remove(tmpfile.Name())
		if removeErr != nil {
			return nil, removeErr
		}

		return nil, fmt.Errorf("Error writing to temporary test file. err: %s", err.Error())
	}

	return tmpfile, nil
}
