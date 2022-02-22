---
layout: "heroku"
page_title: "Heroku: Upgrading the Provider"
sidebar_current: "docs-heroku-guides-upgrading"
description: |-
  Guide to upgrading through major versions.
---

# Upgrading v4 to v5

~> **Major upgrades should be tested with non-critical infrastructure.**

## Summary of changes

The `heroku_app` resource `id` is automatically migrated to contain the app ID (UUID). This means when you reference `heroku_app.example.id`, the UUID is returned, whereas previously this `id` attribute actually contained the app name (surprise!)

All other resources' `heroku_app` reference attributes are now named `app_id`. They only accept UUID values, such as from `heroku_app.example.id`.

All previous `app` attribute values are automatically migrated to `app_id`, performing app lookups to replace names with UUIDs.

## Required configuration changes

Before running `terraform apply` using the new provider version, some manual HCL configuration updates are required:

### 1. Replace resources' `app` attributes with `app_id`

Their existing references to `heroku_app.example.name` must be simultaneously updated to `heroku_app.example.id`.

Previous v4 config example:

```hcl
resource "heroku_addon" "example" {
  app = heroku_app.example.name
  plan = "example-plan"
}
```

Upgraded v5 config example:

```hcl
resource "heroku_addon" "example" {
  app_id = heroku_app.example.id
  plan = "example-plan"
}
```

### 2. Replace `heroku_app` data source result references with `id`

The app data source `id` attribute now contains the app UUID, not the app name. As long as `id` is used for `app_id` references, then no further configuration changes are required. Setting `app_id` references from `name` is no longer allowed, and must be updated to `id`, like `data.heroku_app.example.id`.

## Verify configuration changes

Run `terraform apply` to verify the migration is complete.

## Success look like…

> No changes. Your infrastructure matches the configuration.

Upon the first apply on this new provider version, the state will be automatically updated to contain only UUID-based `id` for app. You may see some changes detected, but they should not result in infrastructure changes.

```
heroku_app.example: Refreshing state... [id=c92d7139-ec41-4088-a71d-3a6dfef99f83]
heroku_addon.example: Refreshing state... [id=41acf474-1b0e-4f53-960f-ca51095b84d7]

Note: Objects have changed outside of Terraform

Terraform detected the following changes made outside of Terraform since the last "terraform apply":

  # heroku_addon.example has been changed
  ~ resource "heroku_addon" "example" {
      + app_id            = "c92d7139-ec41-4088-a71d-3a6dfef99f83"
        id                = "41acf474-1b0e-4f53-960f-ca51095b84d7"
        name              = "deployhooks-silhouetted-32012"
        # (5 unchanged attributes hidden)
    }

…

No changes. Your infrastructure matches the configuration.
```

Once this completes, the state will be migrated! ✅😄

## Failure looks like…

Specific errors about `app` & `app_id` should describe configuration problems, such as:

```
│ Error: Missing required argument
│ 
│   on main.tf line 19, in resource "heroku_addon" "example":
│   19: resource "heroku_addon" "example" {
│ 
│ The argument "app_id" is required, but no definition was found.
```

```
│ Error: Unsupported argument
│ 
│   on main.tf line 20, in resource "heroku_addon" "example":
│   20:   app = heroku_app.example.id
│ 
│ An argument named "app" is not expected here.
```

If it reflects that other changes are detected, possible explanations are:

* missed updating a `heroku_app` reference from `name` to `id`
* infrastructure did change somehow during your work (not related to this migration)
* possibly found a bug in the migration strategy, so please create an [issue on GitHub](https://github.com/heroku/terraform-provider-heroku/issues).
