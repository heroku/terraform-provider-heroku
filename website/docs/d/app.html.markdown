---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-datasource-app-x"
description: |-
  Get information on a Heroku App.
---

# Data Source: heroku_app

Use this data source to get information about a Heroku App.

## Example Usage

```hcl
# Create a new Heroku app
data "heroku_app" "default" {
  name   = "my-cool-app"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application. In Heroku, this is also the
   unique ID, so it must be unique and have a minimum of 3 characters.

## Attributes Reference

The following attributes are exported:

* `name` - (Required) The name of the application. In Heroku, this is also the
   unique .
* `stack` - (Optional) The application stack is what platform to run the application
   in.
* `buildpacks` - (Optional) A list of buildpacks that this app uses.
* `space` - (Optional) The private space in which the app runs. Not present if this is a common runtime app.
* `region` - (Required) The region in which the app is deployed.
* `git_url` - (Required) The Git URL for the application. This is used for
   deploying new versions of the app.
* `web_url` - (Required) The web (HTTP) URL that the application can be accessed
   at by default.
* `heroku_hostname` - (Required) A hostname for the Heroku application, suitable
   for pointing DNS records.
* `config_vars` - (Optional) A map of all of the configuration variables for the app.
* `acm` - (Required) True if Heroku ACM is enabled for this app, false otherwise.
* `organization` - (Optional) The organization that owns this app, if the app is owned by an organization. The fields for this block are documented below.

The `organization` block supports:

* `name` (string) - The name of the organization.
* `locked` (boolean)
* `personal` (boolean)
