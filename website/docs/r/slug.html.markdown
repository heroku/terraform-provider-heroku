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

This resource supports uploading a pre-generated archive file of executable code, making it possible to launch apps directly from a Terraform config. This resource does not itself generate the slug archive. [A guide to creating slug archives](https://devcenter.heroku.com/articles/platform-api-deploying-slugs) is available in the Heroku Devcenter.

## Minimal Example

Create a ready-to-release slug:

~> **NOTE:** 
- When `file_path` is specified, the file it references must contain a slug archive of executable code and must follow the prescribed layout from [Create slug archive](https://devcenter.heroku.com/articles/platform-api-deploying-slugs#create-slug-archive) in the Heroku Devcenter (nested within an `./app` directory)
- When `file_path` is not specified, then the slug archive must be uploaded to the resulting computed `blob.method` + `blob.url` by some other means, otherwise app release will fail with _Compiled slug couldn't be found_
- The archive may be created by an external build system, downloaded from another Heroku app, or otherwise provided outside of the context of this Terraform resource
- If the contents (SHA256) of the file at `file_path` change, then a new resource will be forced on the next plan/apply; if the file does not exist, the difference is ignored.

```hcl
resource "heroku_slug" "foobar" {
  app       = "${heroku_app.foobar.id}"
  // The slug archive file must already exist
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
  // The slug archive file must already exist
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
* `buildpack_provided_description` - Description of language or app framework, `"Ruby/Rack"`; displayed as the app's language in the Heroku Dashboard
* `checksum` - Hash of the slug for verifying its integrity, auto-generated when `file_path` is set to upload a slug archive, `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
* `commit` - Identification of the code with your version control system (eg: SHA of the git HEAD), `"60883d9e8947a57e04dc9124f25df004866a2051"`
* `commit_description` - Description of the provided commit
* `file_path` - Path to a slug archive, see [Create slug archive](https://devcenter.heroku.com/articles/platform-api-deploying-slugs#create-slug-archive) in the Heroku Devcenter, if `file_path` is not specified for a slug, then the slug archive must be uploaded to the computed `blob.method` + `blob.url` by some other means, otherwise an app release with the slug will not be possible because there's no executable code, `"slugs/current.tgz"`
* `process_types` - (Required) Map of [processes to launch on Heroku Dynos](https://devcenter.heroku.com/articles/process-model)
* `stack` - Name or ID of the [Heroku stack](https://devcenter.heroku.com/articles/stack)

## Attributes Reference
The following attributes are exported:
* `id` - The ID of the slug
* `blob` - Slug archive (compressed tar of executable code)
  * `method` - HTTP method to upload the archive
  * `url` - Pre-signed, expiring URL to upload the archive
* `buildpack_provided_description` - Description of language or app framework, `"Ruby/Rack"`
* `checksum` - Hash of the slug for verifying its integrity, auto-generated when `file_path` is set to upload a slug archive, `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
* `commit` - Identification of the code with your version control system (eg: SHA of the git HEAD), `"60883d9e8947a57e04dc9124f25df004866a2051"`
* `commit_description` - Description of the provided commit
* `process_types` - Map of [processes to launch on Heroku Dynos](https://devcenter.heroku.com/articles/process-model)
* `size` - Slug archive filesize in bytes
* `stack` - [Heroku stack](https://devcenter.heroku.com/articles/stack) name
* `stack_id` - [Heroku stack](https://devcenter.heroku.com/articles/stack) ID

## Import
Existing slugs can be imported using the combination of the application name, a colon, and the slug ID.

For example:
```
$ terraform import heroku_slug.foobar bazbux:4f1db8ef-ed5c-4c42-a3d6-3c28262d5abc
```

* `foobar` is the **heroku_slug** resource's name
* `bazbux` is the Heroku app name (or ID) that the slug belongs to
* `:` separates the app identifier & the slug identifier
* `4f1db8ef…` is the slug ID

