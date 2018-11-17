---
layout: "heroku"
page_title: "Heroku: heroku_build"
sidebar_current: "docs-heroku-resource-build"
description: |-
  Provides the ability to build & release code located at an HTTPS URL, making it possible to launch apps directly from a Terraform config
---

# heroku\_build

Provides a [Heroku Build](https://devcenter.heroku.com/articles/platform-api-reference#build)
resource.

## Example Usage

```hcl
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

resource "heroku_build" "foobar" {
  app = "${heroku_app.foobar.id}"
  buildpacks = ["mars/create-react-app"]
  
  source = {
    url     = "https://github.com/mars/cra-example-app/archive/v2.1.1.tar.gz"
    version = "v2.1.1"
  }
}

resource "heroku_formation" "foobar" {
  app        = "${heroku_app.foobar.id}"
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.foobar"]
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The ID of the Heroku app
* `buildpacks` - List of buildpack registry names and/or GitHub URLs
* `source` - (Required) A block that specifies the source code to build & release:
  * `checksum` - Hash of the source archive for verifying its integrity, `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
  * `url` (Required) - Location of the buildpack for the app. Either a GitHub URL or a [buildpack registry name/namespace](https://devcenter.heroku.com/articles/buildpack-registry)
  * `version` - Use to track what version of your source originated this build. If you are creating builds from git-versioned source code, for example, the commit hash or release tag would be a good value to use for the version parameter.


## Attributes Reference

The following attributes are exported:

* `id` - The ID of the build
* `output_stream_url` - URL that [streams the log output from the build](https://devcenter.heroku.com/articles/build-and-release-using-the-api#streaming-build-output)
* `release` - The Heroku app release created with a build's slug
  * `id`
* `slug` - The Heroku slug created by a build
  * `id`
* `stack` - Name or ID of the [Heroku stack](https://devcenter.heroku.com/articles/stack)
* `status` - The status of a build. Possible values are `pending`, `successful` and `failed`
* `user` - Heroku account that created a build
  * `email`
  * `id`

## Import
Existing builds can be imported using the combination of the application name, a colon, and the build ID.

For example:
```
$ terraform import heroku_build.foobar bazbux:4f1db8ef-ed5c-4c42-a3d6-3c28262d5abc
```

* `foobar` is the **heroku_build** resource's name
* `bazbux` is the Heroku app name (or ID) that the build belongs to
* `:` separates the app identifier & the build identifier
* `4f1db8ef…` is the build ID

