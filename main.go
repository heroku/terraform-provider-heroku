package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/heroku/terraform-provider-heroku/v5/heroku"
)

const providerAddr = "registry.terraform.io/heroku/heroku"

func main() {
	ctx := context.Background()

	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	// The provider has been fully migrated to the terraform-plugin-framework
	// (protocol 6); it is served directly with no SDKv2/mux layer.
	err := providerserver.Serve(ctx, heroku.NewFrameworkProvider, providerserver.ServeOpts{
		Address: providerAddr,
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
