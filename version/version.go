package version

//Cribbed from
//https://github.com/terraform-providers/terraform-provider-azurerm/tree/master/version
//This takes advantage of a new build flag populating the binary version of the
//provider, for example:
//-ldflags="-X=github.com/terraform-providers/terraform-provider-heroku/version.ProviderVersion=x.x.x"

var (
	// ProviderVersion is set during the release process to the release version of the binary, and
	// set to acc during tests.
	ProviderVersion = "dev"
)
