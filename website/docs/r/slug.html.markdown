---
layout: "heroku"
page_title: "Heroku: heroku_slug"
sidebar_current: "docs-heroku-resource-slug"
description: |-
  Provides the ability to create & upload a slug (archive of executable code) to an app, making it possible to launch apps directly from a Terraform config
---

# heroku\_slug

Provides a [Heroku Slug](https://devcenter.heroku.com/articles/platform-api-reference#slug)
resource.

This resource supports [Creating Slugs from Scratch](https://devcenter.heroku.com/articles/platform-api-deploying-slugs), making it possible to launch apps directly from a Terraform config.

If `file_path` is not specified for a slug, then the slug archive must be uploaded to `blob.method` + `blob.url` by some other means. Otherwise, a release will not be possible because there's no executable code.

## Minimal Example

All that is required to create a ready-to-release slug:

```hcl
resource "heroku_slug" "foobar" {
  app       = "${heroku_app.foobar.id}"
  file_path = "slug.tgz"

  process_types = {
    web = "ruby server.rb"
  }
}
```

## Example Usage

Complete config to launch a Heroku app:

```hcl
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

# Create a slug for the app with a local slug archive file
resource "heroku_slug" "foobar" {
  app                            = "${heroku_app.foobar.id}"
  buildpack_provided_description = "Ruby"
  file_path                      = "slug.tgz"

  process_types = {
    web = "ruby server.rb"
  }
}

# Deploy a release to the app with the slug
resource "heroku_app_release" "foobar" {
  app     = "${heroku_app.foobar.id}"
  slug_id = "${heroku_slug.foobar.id}"
}

# Launch the app's web process by scaling-up
resource "heroku_formation" "foobar" {
  app        = "${heroku_app.foobar.id}"
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_app_release.foobar"]
}
```

## Argument Reference
* `app` - (Required) The name of the application
* `file_path` - Path to a slug archive, see [Creating Slugs from Scratch](https://devcenter.heroku.com/articles/platform-api-deploying-slugs), `"slugs/current.tgz"`
* `buildpack_provided_description` - Description of language or app framework, `"Ruby/Rack"`; displayed as the app's language in the Heroku Dashboard
* `checksum` - Hash of the slug for verifying its integrity, `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
* `commit` - Identification of the code with your version control system (eg: SHA of the git HEAD), `"60883d9e8947a57e04dc9124f25df004866a2051"`
* `commit_description` - Description of the provided commit
* `process_types` - (Required) Map of [processes to launch on Heroku Dynos](https://devcenter.heroku.com/articles/process-model)
* `stack` - Name or ID of the [Heroku stack](https://devcenter.heroku.com/articles/stack)

## Attributes Reference
The following attributes are exported:
* `id` - The ID of the slug
* `blob` - Slug archive (compressed tar of executable code)
  * `method` - HTTP method to upload the archive
  * `url` - Pre-signed, expiring URL to upload the archive
* `buildpack_provided_description` - Description of language or app framework, `"Ruby/Rack"`
* `checksum` - Hash of the slug for verifying its integrity, `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
* `commit` - Identification of the code with your version control system (eg: SHA of the git HEAD), `"60883d9e8947a57e04dc9124f25df004866a2051"`
* `commit_description` - Description of the provided commit
* `process_types` - Map of [processes to launch on Heroku Dynos](https://devcenter.heroku.com/articles/process-model)
* `size` - Slug archive filesize in bytes
* `stack_id` - [Heroku stack](https://devcenter.heroku.com/articles/stack) identifier
* `stack_name` - [Heroku stack](https://devcenter.heroku.com/articles/stack) name

## Import
Existing slugs can be imported using the combination of the application name, a colon, and the slug ID.

For example:
```
$ terraform import heroku_slug.foobar foobar
```
