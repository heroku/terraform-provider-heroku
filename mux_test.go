package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/heroku/terraform-provider-heroku/v5/heroku"
)

// TestProviderSchema ensures the framework provider serves a valid protocol-6
// provider schema with no error diagnostics. The provider is now served
// directly via providerserver (see main.go); the SDKv2/mux layer was removed
// as part of the migration to terraform-plugin-framework.
func TestProviderSchema(t *testing.T) {
	ctx := context.Background()

	server := providerserver.NewProtocol6(heroku.NewFrameworkProvider())()

	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %s", err)
	}

	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("provider schema diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}

	if resp.Provider == nil {
		t.Fatal("expected a non-nil provider schema")
	}
}
