---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-datasource-app-x"
description: |-
  Get information on a Heroku App.
---

# Data Source: heroku_app

Use this data source to get information about a Heroku app.

This data source supports both the [Cedar and Fir generations](https://devcenter.heroku.com/articles/generations) of Heroku apps.

## Example Usage

```hcl-terraform
# Lookup an existing Heroku app
data "heroku_app" "default" {
  name = "my-cool-app"
}

# Example: Check app generation and buildpacks
output "app_generation" {
  value = data.heroku_app.default.generation
}

output "app_buildpacks" {
  value = data.heroku_app.default.buildpacks
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

* `generation`: The generation of the app platform (`cedar` or `fir`).

* `buildpacks`: The list of buildpacks that this app uses. Empty for apps using Cloud Native Buildpacks, such as Fir-generation apps, which list buildpacks in `project.toml` instead.

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

* `uuid`: The unique UUID of the Heroku app.
