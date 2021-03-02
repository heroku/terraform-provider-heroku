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

```hcl-terraform
# Lookup an existing Heroku app
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

* `id` - The unique UUID of the Heroku app.

* `name` - The name of the application. In Heroku, this is also the
   unique ID.

* `stack` - The application stack is what platform to run the application
   in.

* `buildpacks` - A list of buildpacks that this app uses.

* `space` - The private space in which the app runs. Not present if this is a common runtime app.

* `region` - The region in which the app is deployed.

* `git_url` - The Git URL for the application. This is used for
   deploying new versions of the app.

* `web_url` - The web (HTTP) URL that the application can be accessed
   at by default.

* `heroku_hostname` - The hostname for the Heroku application, suitable
   for pointing DNS records.

* `config_vars` - A map of all configuration variables for the app.

* `acm` - True if Heroku ACM is enabled for this app, false otherwise.

* `organization` - The Heroku Team that owns this app. 

  * `name` - The name of the Heroku Team (organization).

  * `locked` - True if the app access is locked

  * `personal`

* `uuid` - The unique UUID of the Heroku app.