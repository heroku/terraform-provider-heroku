package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/heroku/terraform-provider-heroku/v5/heroku"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: heroku.Provider})
}
