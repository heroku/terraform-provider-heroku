---
layout: "heroku"
page_title: "Heroku: heroku_app_feature"
sidebar_current: "docs-heroku-resource-app-feature"
description: |-
  Provides a resource to create and manage App Features on Heroku.
---

# heroku\_app\_feature

This resource is used to create and manage [App Features](https://devcenter.heroku.com/articles/heroku-beta-features) on Heroku.

## Available Features

For a list of available features, use the [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli) to fetch them for one of your existing apps:

```
heroku labs --app foobar
```

The output will contain **User Features** and **App Features**. This resource manages App Features. If you need to manage User Features, use the [`heroku_account_feature` resource](account_feature.html).

## Example Usage

```hcl
resource "heroku_app_feature" "log_runtime_metrics" {
  app = "ebb78898-5fae-44c1-9a0b-3171e1e4b372"
  name = "log-runtime-metrics"
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) A Heroku app's `UUID`. Can also be the name of the Heroku app but `UUID` is preferred as it is idempotent.
* `name` - (Required) The name of the App Feature to manage.
* `enabled` - (Optional) Whether to enable or disable the App Feature. The default value is true.

## Import
App features can be imported using the combination of the application name, a colon, and the feature's name.

For example:
```
$ terraform import heroku_app_feature.log-runtime-metrics foobar:log-runtime-metrics
```
