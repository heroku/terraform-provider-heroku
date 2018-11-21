---
layout: "heroku"
page_title: "Heroku: heroku_build"
sidebar_current: "docs-heroku-resource-build"
description: |-
  `git push heroku master` for Terraform. Provides the ability to build & release code from a local or remote source archive, making it possible to launch apps directly from a Terraform config
---

# heroku\_build

*`git push heroku master` for Terraform*

Provides a [Heroku Build](https://devcenter.heroku.com/articles/platform-api-reference#build)
resource, to deploy source code to Heroku.

Either a URL or local file path, pointing to a [tarball](https://en.wikipedia.org/wiki/Tar_(computing)) of the source code, may be pushed for a build.

This resource waits until the [build](https://devcenter.heroku.com/articles/build-and-release-using-the-api) & release completes, either succeeds or fails.

When build succeeds, Heroku creates a new app release. [Release phase](https://devcenter.heroku.com/articles/release-phase) will run, if declared for the app. Once release succeeds, the app's dynos will start or restart with the new release.

When build fails, the error will contain a URL to view the build log. `curl "https://the-long-log-url-in-the-error"`.

## Source URLs

A `source.url` may point to any `https://` URL that responds to a `GET` with a tarball source code. When running `terraform apply`, the source code will only be fetched once for a successful build. Change the URL to force a new resource.

### GitHub URLs

GitHub provides tarballs through URLs:

* **release tag** `https://github.com/username/example/archive/v1.0.0.tar.gz`
* **branch name** `https://github.com/username/example/archive/branchname.zip`
* **master** `https://github.com/username/example/archive/master.zip`

## Local source file

A `source.path` may point to a tarball of source code using relative or root paths. When running `terraform apply`, the source path will build if the contents (SHA256) of the file changes since the last `apply`.

## Example Usage with Remote Source

```hcl
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

resource "heroku_build" "foobar" {
  app = "${heroku_app.foobar.id}"
  buildpacks = ["https://github.com/mars/create-react-app-buildpack"]

  source = {
    // This app requires a community buildpack, set it in `buildpacks` above.
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

## Example Usage with Local Source

```hcl
resource "heroku_app" "foobar" {
    name = "foobar"
    region = "us"
}

resource "heroku_build" "foobar" {
  app = "${heroku_app.foobar.id}"

  source = {
    // Apps that use an official Heroku buildpack will be
    // auto-detected. No buildpacks setting required.
    path    = "sources/app-v1.tgz"
    version = "v1"
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
  * `checksum` - Hash of the source archive for verifying its integrity, auto-generated when `source.path` is set, `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
  * `path` - (Required unless `source.url` is set) Local path to the source archive for the app. If the contents (SHA) change
  * `url` - (Required unless `source.path` is set) `https` location of the source archive for the app
  * `version` - Use to track what version of your source originated this build. If you are creating builds from git-versioned source code, for example, the commit hash, or release tag would be a good value to use for the version parameter.


## Attributes Reference

The following attributes are exported:

* `uuid` - The ID of the build
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
