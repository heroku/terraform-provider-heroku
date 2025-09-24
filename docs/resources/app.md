---
layout: "heroku"
page_title: "Heroku: heroku_app"
sidebar_current: "docs-heroku-resource-app-x"
description: |-
  Provides a Heroku App resource. This can be used to create and manage applications on Heroku.
---

# heroku\_app

Provides a Heroku App resource. This can be used to create and manage applications on Heroku.

The Heroku platform supports two generations:
- **Cedar** (default): Traditional platform with support for buildpacks, stack configuration, and internal routing
- **Fir**: Next-generation platform with Cloud Native Buildpacks (CNB), enhanced security, and modern containerization

-> **Always reference apps by ID (UUID) in Terraform configuration**
Starting with v5.0 of this provider, all HCL app references are by ID. Read more details in [Upgrading](guides/upgrading.html).

## Example Usage

### Cedar Generation Using Classic Buildpacks (Default)
```hcl-terraform
resource "heroku_app" "cedar_app" {
  name   = "my-cedar-app"
  region = "us"

  config_vars = {
    FOOBAR = "baz"
  }

  buildpacks = [
    "heroku/go"
  ]
  
  stack = "heroku-22"
}
```

### Fir Generation Using Cloud Native Buildpacks (via Fir Space)
```hcl-terraform
# Create a Fir generation space first
resource "heroku_space" "fir_space" {
  name         = "my-fir-space"
  organization = "my-org"
  region       = "virginia"
  generation   = "fir"
}

# Apps deployed to Fir spaces automatically use Fir generation
resource "heroku_app" "fir_app" {
  name   = "my-fir-app"
  region = "virginia"
  space  = heroku_space.fir_space.name

  organization {
    name = "my-org"
  }

  config_vars = {
    FOOBAR = "baz"
  }
  
  # Note: buildpacks and stack are not supported for Fir generation
  # Use project.toml in your application code instead
}
```

## Example Usage for a Team

A Heroku "team" was originally called an "organization", and that is still the identifier used in this resource.

```hcl-terraform
resource "heroku_app" "default" {
  name   = "my-cool-app"
  region = "us"

  organization {
    name = "my-cool-team"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application. In Heroku, this is also the
   unique ID, so it must be unique and have a minimum of 3 characters.
* `region` - (Required) The region that the app should be deployed in.
* `generation` - (Computed) Generation of the app platform. Automatically determined based on the space the app is deployed to. Apps in Fir generation spaces will be `fir`, all other apps will be `cedar`.
   - `cedar`: Traditional platform supporting buildpacks, stack configuration, and internal routing
   - `fir`: Next-generation platform with Cloud Native Buildpacks (CNB). Does not support `buildpacks`, `stack`, or `internal_routing` fields
* `stack` - (Optional) The application stack is what platform to run the application in. **Note**: Not supported for `fir` generation apps.
* `buildpacks` - (Optional) Buildpack names or URLs for the application.
  Buildpacks configured externally won't be altered if this is not present. **Note**: Not supported for `fir` generation apps. Use project.toml for Cloud Native Buildpacks configuration instead.
* `config_vars`<sup>[1](#deleting-vars)</sup> - (Optional) Configuration variables for the application.
     The config variables in this map are not the final set of configuration
     variables, but rather variables you want present. That is, other
     configuration variables set externally won't be removed by Terraform
     if they aren't present in this list.
* `sensitive_config_vars`<sup>[1](#deleting-vars)</sup> - (Optional) This argument is the same as `config_vars`.
     The main difference between the two is when `sensitive_config_vars` outputs
     are displayed on-screen following a terraform apply or terraform refresh,
     they are redacted, with <sensitive> displayed in place of their value.
     It is recommended to put private keys, passwords, etc in this argument.
* `space` - (Optional) The name of a private space to create the app in.
* `internal_routing` - (Optional) If true, the application will be routable
  only internally in a private space. This option is only available for apps
  that also specify `space`. **Note**: Not supported for `fir` generation apps.
* `organization` - (Optional) A block that can be specified once to define
     Heroku Team settings for this app. The fields for this block are
     documented below.
* `acm` - (Optional) The flag representing Automated Certificate Management for the app.

The `organization` block supports:
* `name` (string) - The name of the Heroku Team.
* `locked` (boolean) - Are other team members forbidden from joining this app.
* `personal` (boolean) - Force creation of the app in the user account even if a default team is set.

### Deleting vars

Deleting an entire `config_vars` or `sensitive_config_vars` map from a `heroku_app`
configuration will not actually remove the vars on the remote resource. To remove an existing variable,
leave these attribute maps in-place and delete only its entries from the map. Once these attributes are
empty, the map itself may be deleted from the configuration. Otherwise if one deletes the map with existing
entries, the config vars will not be deleted from the remote resource.

This is especially important if you are migrating all `config_vars` to `sensitive_config_vars` or migrating
config vars to `heroku_app_config_association` resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID (UUID) of the app.
* `name` - The name of the app.
* `generation` - Generation of the app platform (cedar or fir). Automatically determined from the space the app is deployed to.
* `stack` - The application stack is what platform to run the application in.
* `space` - The private space the app should run in.
* `internal_routing` - Whether internal routing is enabled the private space app.
* `region` - The region that the app should be deployed in.
* `git_url` - The Git URL for the application. This is used for
   deploying new versions of the app.
* `web_url` - The web (HTTP) URL that the application can be accessed
   at by default.
* `heroku_hostname` - A hostname for the Heroku application, suitable
   for pointing DNS records.
* `all_config_vars` - A map of all configuration variables that
  exist for the app, containing both those set by Terraform and those
  set externally. (These are treated as "sensitive" so that
  their values are redacted in console output.) This attribute is not set in state if the `provider`
  attribute `set_app_all_config_vars_in_state` is `false`.
* `uuid` - The unique UUID of the Heroku app. **NOTE:** Use this for `null_resource` triggers.

## Cloud Native Buildpacks (Fir Generation)

When apps are deployed to Fir generation spaces, they automatically use Cloud Native Buildpacks (CNB) instead of traditional Heroku buildpacks. This requires different configuration approaches:

### project.toml Configuration

Instead of specifying `buildpacks` in Terraform, create a `project.toml` file in your application root:

```toml
[build]
[[build.buildpacks]]
id = "heroku/nodejs"

[[build.buildpacks]] 
id = "heroku/procfile"

[build.env]
BP_NODE_VERSION = "18.*"
```

### Migration from Cedar to Fir

When migrating from Cedar to Fir generation:

1. **Create Fir space**: Create a new space with `generation = "fir"`
2. **Remove incompatible fields**: Remove `buildpacks`, `stack`, and `internal_routing` from your Terraform configuration
3. **Add project.toml**: Create a `project.toml` file in your application code with buildpack configuration
4. **Update app space**: Change your app's `space` to use the Fir space
5. **Redeploy**: Deploy your application with the new configuration

```hcl-terraform
# Before (Cedar)
resource "heroku_space" "cedar_space" {
  name         = "my-space"
  organization = "my-org"
  region       = "virginia"
}

resource "heroku_app" "example" {
  name   = "my-app"
  region = "virginia"
  space  = heroku_space.cedar_space.name
  
  buildpacks = ["heroku/nodejs"]
  stack      = "heroku-22"
}

# After (Fir)
resource "heroku_space" "fir_space" {
  name         = "my-space-fir"
  organization = "my-org"
  region       = "virginia" 
  generation   = "fir"
}

resource "heroku_app" "example" {
  name   = "my-app"
  region = "virginia"
  space  = heroku_space.fir_space.name
  
  # buildpacks and stack removed - configured via project.toml
  # generation is automatically "fir" from the space
}
```

## Import

Apps can be imported using an existing app's `UUID` or name.

For example:
```
$ terraform import heroku_app.foobar MyApp
$ terraform import heroku_app.foobar e74ac056-7d00-4a7e-aa80-df4bc413a825
```

Please note: `config_vars` & `sensitive_config_vars` will not be imported due to limitations of Terraform's import process (see [issue](https://github.com/heroku/terraform-provider-heroku/issues/247#issuecomment-602013774)). All vars will appear to be added on the next plan/apply. The diff may be manually reconciled using the outputs of `heroku config` & `terraform plan`.
