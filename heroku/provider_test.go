package heroku

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	heroku "github.com/heroku/heroku-go/v3"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var testAccConfig *TestConfig

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"heroku": testAccProvider,
	}
	testAccConfig = NewTestConfig()
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

type TestConfigKey int

const (
	TestConfigUserKey TestConfigKey = iota
	TestConfigAcceptanceTestKey
	TestConfigNonAdminUserKey
	TestConfigAPIKey
	TestConfigOrganizationKey
	TestConfigSpaceOrganizationKey
	TestConfigSlugIDKey
)

var testConfigKeyToEnvName = map[TestConfigKey]string{
	TestConfigUserKey:              "HEROKU_TEST_USER",
	TestConfigNonAdminUserKey:      "HEROKU_NON_ADMIN_TEST_USER",
	TestConfigAPIKey:               "HEROKU_API_KEY",
	TestConfigOrganizationKey:      "HEROKU_ORGANIZATION",
	TestConfigSpaceOrganizationKey: "HEROKU_SPACES_ORGANIZATION",
	TestConfigSlugIDKey:            "HEROKU_SLUG_ID",
	TestConfigAcceptanceTestKey:    resource.TestEnvVar,
}

func (k TestConfigKey) String() (name string) {
	if val, ok := testConfigKeyToEnvName[k]; ok {
		name = val
	}
	return
}

type TestConfig struct{}

func NewTestConfig() *TestConfig {
	return &TestConfig{}
}

func (t *TestConfig) Get(keys ...TestConfigKey) (val string) {
	for _, key := range keys {
		val = os.Getenv(key.String())
		if val != "" {
			break
		}
	}
	return
}

func (t *TestConfig) GetOrSkip(testing *testing.T, keys ...TestConfigKey) (val string) {
	t.SkipUnlessAccTest(testing)
	val = t.Get(keys...)
	if val == "" {
		testing.Skip(fmt.Sprintf("skipping test: config %v not set", keys))
	}
	return
}

func (t *TestConfig) GetOrAbort(testing *testing.T, keys ...TestConfigKey) (val string) {
	t.SkipUnlessAccTest(testing)
	val = t.Get(keys...)
	if val == "" {
		testing.Fatal(fmt.Sprintf("stopping test: config %v must be set", keys))
	}
	return
}

func (t *TestConfig) SkipUnlessAccTest(testing *testing.T) {
	val := t.Get(TestConfigAcceptanceTestKey)
	if val == "" {
		testing.Skip(fmt.Sprintf("Acceptance tests skipped unless env '%s' set", TestConfigAcceptanceTestKey.String()))
	}
}

func (t *TestConfig) GetAnyOrganizationOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigSpaceOrganizationKey, TestConfigOrganizationKey)
}

func (t *TestConfig) GetNonAdminUserOrAbort(testing *testing.T) (val string) {
	return t.GetOrAbort(testing, TestConfigNonAdminUserKey)
}

func (t *TestConfig) GetOrganizationOrAbort(testing *testing.T) (val string) {
	return t.GetOrAbort(testing, TestConfigOrganizationKey)
}

func (t *TestConfig) GetOrganizationOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigOrganizationKey)
}

func (t *TestConfig) GetSlugIDOrAbort(testing *testing.T) (val string) {
	return t.GetOrAbort(testing, TestConfigSlugIDKey)
}

func (t *TestConfig) GetSlugIDOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigSlugIDKey)
}

func (t *TestConfig) GetSpaceOrganizationOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigSpaceOrganizationKey)
}

func (t *TestConfig) GetUserOrAbort(testing *testing.T) (val string) {
	return t.GetOrAbort(testing, TestConfigUserKey)
}

func (t *TestConfig) GetUserOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigUserKey)
}

func testAccPreCheck(t *testing.T) {
	testAccConfig.GetOrAbort(t, TestConfigAPIKey)
}
