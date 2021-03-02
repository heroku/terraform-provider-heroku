---
layout: "heroku"
page_title: "Heroku: heroku_build"
sidebar_current: "docs-heroku-resource-build"
description: |-
  "Deploy to Heroku" for Terraform. Provides the ability to build & release code from a local or remote source,
  making it possible to launch apps directly from a Terraform config
---

# heroku\_build

Provides a [Heroku Build](https://devcenter.heroku.com/articles/platform-api-reference#build)
resource, to deploy source code to a Heroku app.

Either a [URL](#source-urls) or [local path](#local-source), pointing to a [tarball](https://en.wikipedia.org/wiki/Tar_(computing))
of the source code, may be deployed. If a local path is used, it may instead point to a directory of source code, which will be tarballed automatically and then deployed.

This resource waits until the [build](https://devcenter.heroku.com/articles/build-and-release-using-the-api)
& [release](https://devcenter.heroku.com/articles/release-phase) completes.

If the build fails, the error will contain a URL to view the build log. `curl "https://the-long-log-url-in-the-error"`.

To start the app from a successful build, use a [Formation resource](formation.html) to specify the process, dyno size, and dyno quantity.

## Source code layout

The code contained in the source directory or tarball must follow the layout required by the [buildpack](https://devcenter.heroku.com/articles/buildpacks)
or the `Dockerfile` for [container builds](https://devcenter.heroku.com/articles/build-docker-images-heroku-yml).

### Building with Buildpacks

This is the default build process.

For apps that do not have a buildpack set, the [official Heroku buildpacks](https://devcenter.heroku.com/articles/buildpacks#officially-supported-buildpacks)
will be searched until a match is detected and used to compile the app.

A [`Procfile`](https://devcenter.heroku.com/articles/procfile) may be required to successfully launch the app.
Some buildpacks provide a default web process, such as [`npm start` for Node.js](https://devcenter.heroku.com/articles/nodejs-support#default-web-process-type).
Other buildpacks may require a `Procfile`, like for a [pure Ruby app](https://devcenter.heroku.com/articles/ruby-support#ruby-applications-process-types).

### Building with Docker

To use container builds, set the parent `heroku_app` resource's `stack = "container"`

A [`heroku.yml` manifest](https://devcenter.heroku.com/articles/build-docker-images-heroku-yml#heroku-yml-overview)
file is required to declare which `Dockerfile` to build for each process. Be careful not to create conflicting configuration
between `heroku.yml` and Terraform, such as addons or config vars.

## Source URLs
A `source.url` may point to any `https://` URL that responds to a `GET` with a tarball source code. When running `terraform apply`,
the source code will only be fetched once for a successful build. Change the URL to force a new resource.

Useful for building public, open-source source code, such as projects that publish releases on GitHub.

### GitHub URLs
GitHub provides [release](https://help.github.com/articles/creating-releases/) tarballs through URLs. Create a release
and then use the tag as a `source.url`, such as:

```
https://github.com/username/example/archive/v1.0.0.tar.gz
```

Using a branch or master `source.url` is possible, but be aware that tracking down exactly what commit was deployed
for a given `terraform apply` may be difficult. On the other hand, using stable release tags ensures repeatability
of the Terraform configuration.

### Example Usage with Source URL

```hcl-terraform
resource "heroku_app" "foobar" {
    name   = "foobar"
    region = "us"
}

resource "heroku_build" "foobar" {
  app        = heroku_app.foobar.id
  buildpacks = ["https://github.com/mars/create-react-app-buildpack"]

  source {
    # This app uses a community buildpack, set it in `buildpacks` above.
    url     = "https://github.com/mars/cra-example-app/archive/v2.1.1.tar.gz"
    version = "v2.1.1"
  }
}

resource "heroku_formation" "foobar" {
  app        = heroku_app.foobar.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.foobar"]
}
```

## Local source
A `source.path` may point to either:

* a tarball of source code
* a directory of source code
  * use `src/appname` relative paths to child directories within the Terraform project repo (recommended)
  * use `/opt/src/appname` absolute or `../appname` relative paths to external directories
  * **avoid ancestor paths that contain the Terraform configuration itself**
    * paths such as `../` will [cause errors during apply](https://github.com/heroku/terraform-provider-heroku/issues/269)

When running `terraform apply`, if the contents (SHA256) of the source path changed since the last `apply`, then a new build will start.

### Example Usage with Local Source Directory

```hcl-terraform
resource "heroku_app" "foobar" {
    name   = "foobar"
    region = "us"
}

resource "heroku_build" "foobar" {
  app = heroku_app.foobar.id

  source {
    # A local directory, changing its contents will
    # force a new build during `terraform apply`
    path = "src/example-app"
  }
}

resource "heroku_formation" "foobar" {
  app        = heroku_app.foobar.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.foobar"]
}
```

## Argument Reference

The following arguments are supported:

* `app` - (Required) The ID of the Heroku app
* `buildpacks` - List of buildpack GitHub URLs
* `source` - (Required) A block that specifies the source code to build & release:
  * `checksum` - Hash of the source archive for verifying its integrity, auto-generated when `source.path` is set,
    `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
  * `path` - (Required unless `source.url` is set) Local path to the source directory or tarball archive for the app
  * `url` - (Required unless `source.path` is set) `https` location of the source archive for the app
  * `version` - Use to track what version of your source originated this build. If you are creating builds
    from git-versioned source code, for example, the commit hash, or release tag would be a good value to use for the
    version parameter.


## Attributes Reference

The following attributes are exported:

* `uuid` - The ID of the build
* `output_stream_url` - URL that [streams the log output from the build](https://devcenter.heroku.com/articles/build-and-release-using-the-api#streaming-build-output)
* `release_id` - The Heroku app release created with a build's slug
* `slug_id` - The Heroku slug created by a build
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
