package heroku

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	TestConfigNonAdminUserKey
	TestConfigAPIKey
	TestConfigOrganizationKey
	TestConfigSpaceOrganizationKey
)

var testConfigKeyToEnvName = map[TestConfigKey]string{
	TestConfigUserKey:              "HEROKU_TEST_USER",
	TestConfigNonAdminUserKey:      "HEROKU_NON_ADMIN_TEST_USER",
	TestConfigAPIKey:               "HEROKU_API_KEY",
	TestConfigOrganizationKey:      "HEROKU_ORGANIZATION",
	TestConfigSpaceOrganizationKey: "HEROKU_SPACES_ORGANIZATION",
}

func (k TestConfigKey) String() (name string) {
	if val, ok := testConfigKeyToEnvName[k]; ok {
		name = val
	}
	return
}

type TestConfigKeyNotFoundAction int

const (
	TestConfigKeyNotFoundNoopAction TestConfigKeyNotFoundAction = iota
	TestConfigKeyNotFoundSkipAction
	TestConfigKeyNotFoundAbortAction
)

type TestConfig struct{}

func NewTestConfig() *TestConfig {
	return &TestConfig{}
}

func (t *TestConfig) GetWithAction(key TestConfigKey, testing *testing.T, notFoundAction TestConfigKeyNotFoundAction, defaultValue ...string) (val string) {
	val = os.Getenv(key.String())
	if val == "" && len(defaultValue) > 0 {
		val = defaultValue[0]
	}
	if val == "" && testing != nil {
		switch notFoundAction {
		case TestConfigKeyNotFoundSkipAction:
			testing.Skip(fmt.Sprintf("skipping test: config %s not set", key))
		case TestConfigKeyNotFoundAbortAction:
			testing.Fatal(fmt.Sprintf("stopping test: config %s must be set", key))
		}
	}
	return
}

func (t *TestConfig) Get(key TestConfigKey, testing *testing.T, defaultValue ...string) (val string) {
	return t.GetWithAction(key, testing, TestConfigKeyNotFoundNoopAction, defaultValue...)
}

func (t *TestConfig) GetOrSkip(key TestConfigKey, testing *testing.T, defaultValue ...string) (val string) {
	return t.GetWithAction(key, testing, TestConfigKeyNotFoundSkipAction, defaultValue...)
}

func (t *TestConfig) GetOrAbort(key TestConfigKey, testing *testing.T, defaultValue ...string) (val string) {
	return t.GetWithAction(key, testing, TestConfigKeyNotFoundAbortAction, defaultValue...)
}

func (t *TestConfig) GetSpaceOrganizationOrSkip(testing *testing.T) (val string) {
	return t.GetWithAction(TestConfigSpaceOrganizationKey, testing, TestConfigKeyNotFoundSkipAction, t.Get(TestConfigOrganizationKey, testing))
}

func (t *TestConfig) GetNonAdminUserOrAbort(testing *testing.T) (val string) {
	return t.GetWithAction(TestConfigNonAdminUserKey, testing, TestConfigKeyNotFoundAbortAction)
}

func (t *TestConfig) GetUserOrSkip(testing *testing.T) (val string) {
	return t.GetWithAction(TestConfigUserKey, testing, TestConfigKeyNotFoundSkipAction)
}

func getTestUser() string {
	return testAccConfig.Get(TestConfigUserKey, nil)
}

func getTestSpaceOrganizationName() string {
	return testAccConfig.Get(TestConfigSpaceOrganizationKey, nil, testAccConfig.Get(TestConfigOrganizationKey, nil))
}

func testAccPreCheck(t *testing.T) {
	testAccConfig.GetOrAbort(TestConfigAPIKey, t)
}
