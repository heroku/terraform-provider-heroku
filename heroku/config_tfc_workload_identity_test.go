package heroku

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProviderConfigureTFCWorkloadIdentity(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		raw      map[string]interface{}
		wantAuth string
	}{
		{"token used when no api key", map[string]string{"TFC_WORKLOAD_IDENTITY_TOKEN": "tfc-jwt"}, nil, "Bearer tfc-jwt"},
		{"explicit api_key wins", map[string]string{"TFC_WORKLOAD_IDENTITY_TOKEN": "tfc-jwt"}, map[string]interface{}{"api_key": "key"}, "Basic"},
		{"HEROKU_API_KEY wins", map[string]string{"TFC_WORKLOAD_IDENTITY_TOKEN": "tfc-jwt", "HEROKU_API_KEY": "key"}, nil, "Basic"},
		{"opt-out attribute", map[string]string{"TFC_WORKLOAD_IDENTITY_TOKEN": "tfc-jwt"}, map[string]interface{}{"tfc_workload_identity": false}, ""},
		{"opt-out env", map[string]string{"TFC_WORKLOAD_IDENTITY_TOKEN": "tfc-jwt", "HEROKU_TFC_WORKLOAD_IDENTITY": "false"}, nil, ""},
		{"no token", nil, nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NETRC_PATH", filepath.Join(t.TempDir(), "none"))
			for _, k := range []string{"HEROKU_API_KEY", "HEROKU_EMAIL", "TFC_WORKLOAD_IDENTITY_TOKEN", "HEROKU_TFC_WORKLOAD_IDENTITY"} {
				t.Setenv(k, tc.env[k])
			}

			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.Write([]byte(`{"name":"some-app"}`))
			}))
			defer srv.Close()

			raw := map[string]interface{}{"url": srv.URL}
			for k, v := range tc.raw {
				raw[k] = v
			}
			d := schema.TestResourceDataRaw(t, Provider().Schema, raw)

			client, err := providerConfigure(d)
			if err != nil {
				t.Fatal(err)
			}
			config := client.(*Config)
			if s := config.String(); strings.Contains(s, "tfc-jwt") {
				t.Errorf("Config.String() leaks token: %s", s)
			}
			if _, err := config.Api.AppInfo(context.Background(), "x"); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(gotAuth, tc.wantAuth) || (tc.wantAuth == "" && gotAuth != "") {
				t.Errorf("Authorization = %q, want prefix %q", gotAuth, tc.wantAuth)
			}
		})
	}
}
