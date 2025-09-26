---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-datasource-app-x"
description: |-
  Get information on a Heroku App.
---

# Data Source: heroku_app

Use this data source to get information about a Heroku app.

~> **NOTE:** This resource is only supported for apps that use [classic buildpacks](https://devcenter.heroku.com/articles/buildpacks#classic-buildpacks).

## Example Usage

```hcl-terraform
# Lookup an existing Heroku app
data "heroku_app" "default" {
  name   = "my-cool-app"
}
```

## Argument Reference

The following arguments are supported:

* `name`: (Required) The name of the application. In Heroku, it's a
   unique ID, so it must be unique and have a minimum of 3 characters.

## Attributes Reference

The following attributes are exported:

* `id`: The unique UUID of the Heroku app.

* `name`: The name of the application. In Heroku, it's also the
   unique ID.

* `stack`: The application [stack](https://devcenter.heroku.com/articles/stack) is what platform to run the application in.

* `buildpacks`: A list of buildpacks that this app uses.

* `space`: The [space](https://devcenter.heroku.com/articles/private-spaces) in which the app runs. Not present for [Common Runtime](https://devcenter.heroku.com/articles/dyno-runtime#common-runtime) apps.

* `region`: The region in the app is deployed in.

* `git_url`: The Git URL for the application, used for
   deploying new versions of the app.

* `web_url`: The web (HTTP) URL to access the application at by default.

* `heroku_hostname`: The hostname for the Heroku application, suitable
   for pointing DNS records.

* `last_release_id`: The last successful Release ID for the app. May be empty.

* `last_slug_id`: The slug ID from the last successful release. May be empty.

* `config_vars`: The map of all configuration variables for the app.

* `acm`: True if [Heroku Automated Certificate Management](https://devcenter.heroku.com/articles/automated-certificate-management) is enabled for this app, false otherwise.

* `organization`: The Heroku team that owns this app. 

  * `name`: The name of the Heroku team (organization).

  * `locked`: True if the app access is locked

  * `personal`: True for personal apps

* `uuid` - The unique UUID of the Heroku app.
