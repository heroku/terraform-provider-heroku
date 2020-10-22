package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/heroku/terraform-provider-heroku/v3/heroku"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: heroku.Provider})
}
