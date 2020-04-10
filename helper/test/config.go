package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

type TestConfigKey int

const (
	TestConfigUserKey TestConfigKey = iota
	TestConfigAcceptanceTestKey
	TestConfigNonAdminUserKey
	TestConfigAPIKey
	TestConfigOrganizationKey
	TestConfigSpaceOrganizationKey
	TestConfigSlugIDKey
	TestConfigEmail
	TestConfigTeam
	TestConfigUserID
)

var testConfigKeyToEnvName = map[TestConfigKey]string{
	TestConfigUserKey:              "HEROKU_TEST_USER",
	TestConfigNonAdminUserKey:      "HEROKU_NON_ADMIN_TEST_USER",
	TestConfigAPIKey:               "HEROKU_API_KEY",
	TestConfigOrganizationKey:      "HEROKU_ORGANIZATION",
	TestConfigSpaceOrganizationKey: "HEROKU_SPACES_ORGANIZATION",
	TestConfigSlugIDKey:            "HEROKU_SLUG_ID",
	TestConfigEmail:                "HEROKU_EMAIL",
	TestConfigTeam:                 "HEROKU_TEAM",
	TestConfigUserID:               "HEROKU_USER_ID",
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

func (t *TestConfig) GetEmailOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigEmail)
}

func (t *TestConfig) GetTeamOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigTeam)
}

func (t *TestConfig) GetUserIDOrSkip(testing *testing.T) (val string) {
	return t.GetOrSkip(testing, TestConfigUserID)
}
