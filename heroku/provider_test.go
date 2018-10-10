package heroku

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	helper "github.com/terraform-providers/terraform-provider-heroku/helper/test"
	"io/ioutil"
	"os"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var testAccConfig *helper.TestConfig

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"heroku": testAccProvider,
	}
	testAccConfig = helper.NewTestConfig()
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

	c := client.(*Config).Api
	c.URL = srv.URL

	_, err = c.AppInfo(context.Background(), "does-not-matter")
	if err != nil {
		t.Fatal(err)
	}
}

// TODO: uncomment when a better to test netrc isolated from env
//func TestProviderConfigureUseNetrc(t *testing.T) {
//	// Create a dummy netrc file
//	tmpfileNetrc, err := createTempConfigFile(`machine api.heroku.com login email_login password api_key`, ".netrc")
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	defer os.Remove(tmpfileNetrc.Name())
//	os.Setenv("NETRC_PATH", tmpfileNetrc.Name())
//	defer os.Unsetenv("NETRC_PATH")
//	raw := make(map[string]interface{})
//	rawConfig, err := config.NewRawConfig(raw)
//	if err != nil {
//		t.Fatalf("Error creating mock config: %s", err.Error())
//	}
//
//	rp := Provider()
//	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
//	meta := rp.(*schema.Provider).Meta()
//	if meta == nil {
//		t.Fatalf("Expected metadata, got nil. err: %s", err.Error())
//	}
//	configuration := meta.(*Config)
//
//	assert.Equal(t, "email_login", configuration.Email)
//	assert.Equal(t, "api_key", configuration.APIKey)
//}

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
		os.Remove(tmpfile.Name())
		return nil, fmt.Errorf("Error writing to temporary test file. err: %s", err.Error())
	}

	return tmpfile, nil
}
