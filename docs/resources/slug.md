---
layout: "heroku"
page_title: "Heroku: heroku_slug"
sidebar_current: "docs-heroku-resource-slug"
description: |-
  Provides the ability to create & upload a slug (archive of executable code) to an app,
  making it possible to launch apps directly from a Terraform config
---

# heroku\_slug

Provides a [Heroku Slug](https://devcenter.heroku.com/articles/platform-api-reference#slug)
resource.

This resource supports uploading a pre-generated archive file of executable code, making it possible to launch apps
directly from a Terraform config. This resource doesn't generate the slug archive itself.
[A guide to creating slug archives](https://devcenter.heroku.com/articles/platform-api-deploying-slugs) is available
in the Heroku Dev Center.

~> **NOTE:** This resource is only supported for apps that use [classic buildpacks](https://devcenter.heroku.com/articles/buildpacks#classic-buildpacks).

## Minimal Example

Create a ready-to-release slug:

* `file_url` or `file_path` must reference a file containing a slug archive of executable code
  and must follow the prescribed layout from [Create slug archive](https://devcenter.heroku.com/articles/platform-api-deploying-slugs#create-slug-archive)
  in the Heroku Dev Center (nested within an `./app` directory)
* The archive can be created by an external build system, downloaded from another Heroku app,
  or otherwise provided outside of the context of this Terraform resource
* If the content (SHA256) of `file_path` changes, then a new resource is forced on the next plan/apply;
  if the file doesn't exist, the difference is ignored.
* The `file_url` is only fetched during resource creation. To trigger another fetch, change the `file_url`,
  then a new resource is forced on the next plan/apply.

```hcl-terraform
resource "heroku_app" "foobar" {
  name = "foobar"
  region = "us"
}

resource "heroku_slug" "foobar" {
  app_id   = heroku_app.foobar.id
  file_url = "https://github.com/heroku/terraform-provider-heroku/raw/master/heroku/test-fixtures/slug.tgz"

  process_types = {
    web = "ruby server.rb"
  }
}
```

## Example Usage

Complete config to launch a Heroku app:

```hcl-terraform
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

# Create a slug for the app with a local slug archive file
resource "heroku_slug" "foobar" {
  app_id                         = heroku_app.foobar.id
  buildpack_provided_description = "Ruby"
  // The slug archive file must already exist
  file_path                      = "slug.tgz"

  process_types = {
    web = "ruby server.rb"
  }
}

# Deploy a release to the app with the slug
resource "heroku_app_release" "foobar" {
  app_id  = heroku_app.foobar.id
  slug_id = heroku_slug.foobar.id
}

# Launch the app's web process by scaling-up
resource "heroku_formation" "foobar" {
  app_id     = heroku_app.foobar.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_app_release.foobar"]
}
```

## Argument Reference

The following arguments are supported:

* `app_id`: (Required) The Heroku app ID (not name)
* `buildpack_provided_description` - The description of language or app framework, `"Ruby/Rack"`;
  displayed as the app's language in the Heroku Dashboard
* `checksum` - The hash of the slug for verifying its integrity, auto-generated from contents of `file_path` or `file_url`,
  `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
* `commit` - The identification of the code with your version control system (example: SHA of the git HEAD), `"60883d9e8947a57e04dc9124f25df004866a2051"`
* `commit_description` - The description of the provided commit
* `file_path` - (Required unless `file_url` is set) The local path to the slug archive, `"slugs/current.tgz"`
* `file_url` - (Required unless `file_path` is set) The **https** URL to the slug archive, `"https://example.com/slugs/app-v1.tgz"`
* `process_types` - (Required) The map of [processes to launch on Heroku Dynos](https://devcenter.heroku.com/articles/process-model)
* `stack` - The name or ID of the [Heroku stack](https://devcenter.heroku.com/articles/stack)

## Attributes Reference

The following attributes are exported:

* `id`: The ID of the slug
* `app`: The ID or unique name of the Heroku app
* `blob`: The slug archive (compressed tar of executable code)
  * `method`: The HTTP method to upload the archive
  * `url`: The pre-signed, expiring URL to upload the archive
* `buildpack_provided_description`: The description of language or app framework, `"Ruby/Rack"`
* `checksum`: The hash of the slug for verifying its integrity, auto-generated from contents of `file_path` or `file_url`
* `commit`: The identification of the code with your version control system (example: SHA of the git HEAD), `"60883d9e8947a57e04dc9124f25df004866a2051"`
* `commit_description`: The description of the provided commit
* `process_types`: The map of [processes to launch on Heroku Dynos](https://devcenter.heroku.com/articles/process-model)
* `size`: The slug archive filesize in bytes
* `stack`: The [Heroku stack](https://devcenter.heroku.com/articles/stack) name
* `stack_id`: The [Heroku stack](https://devcenter.heroku.com/articles/stack) ID

## Import
Import existing slugs with the combination of the application name, a colon, and the slug ID.

For example:

```
$ terraform import heroku_slug.foobar bazbux:4f1db8ef-ed5c-4c42-a3d6-3c28262d5abc
```

* `foobar` is the **heroku_slug** resource's name
* `bazbux` is the Heroku app name (or ID) that the slug belongs to
* `:` separates the app identifier & the slug identifier
* `4f1db8ef…` is the slug ID

