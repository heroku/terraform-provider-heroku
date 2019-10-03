package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/terraform-providers/terraform-provider-heroku/heroku"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: heroku.Provider})
}
