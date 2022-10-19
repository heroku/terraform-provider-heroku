package heroku

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	helper "github.com/heroku/terraform-provider-heroku/v5/helper/test"
)

const (
	ProviderNameHeroku = "heroku"
)

var providers []*schema.Provider
var testAccProviderFactories map[string]func() (*schema.Provider, error)
var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var testAccConfig *helper.TestConfig

func testAccProviderFactoriesInternal(providers []*schema.Provider) map[string]func() (*schema.Provider, error) {
	return testAccProviderFactoriesInit(providers, []string{ProviderNameHeroku})
}

// testAccProviderFactoriesInit creates ProviderFactories for the provider under testing.
func testAccProviderFactoriesInit(providers []*schema.Provider, providerNames []string) map[string]func() (*schema.Provider, error) {
	var factories = make(map[string]func() (*schema.Provider, error), len(providerNames))

	for _, name := range providerNames {
		p := Provider()

		factories[name] = func() (*schema.Provider, error) {
			return p, nil
		}

		if providers != nil {
			providers = append(providers, p)
		}
	}

	return factories
}

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"heroku": testAccProvider,
	}
	testAccProviderFactories = testAccProviderFactoriesInit(providers, []string{ProviderNameHeroku})
	testAccConfig = helper.NewTestConfig()
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderConfigureUsesHeadersForClient(t *testing.T) {
	p := Provider()
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

		_, writeErr := w.Write([]byte(`{"name":"some-app"}`))
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}))
	defer srv.Close()

	c := client.(*Config).Api
	c.URL = srv.URL

	_, err = c.AppInfo(context.Background(), "does-not-matter")
	if err != nil {
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
