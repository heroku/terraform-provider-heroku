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

You can either deploy a [URL](#source-urls) or [local path](#local-source), pointing to a [tarball](https://en.wikipedia.org/wiki/Tar_(computing))
of the source code. If you use a local path, it can instead point to a directory of source code, which will be tarballed automatically and then deployed.

This resource waits until the [build](https://devcenter.heroku.com/articles/build-and-release-using-the-api)
and [release](https://devcenter.heroku.com/articles/release-phase) completes.

If the build fails, the build log is output in the error message.

To start the app from a successful build, use a [Formation resource](formation.html) to specify the process, dyno size, and dyno quantity.

## Buildpack Configuration

Build configuration varies between apps that use [classic buildpacks vs. Cloud Native Buildpacks (CNBs)](https://devcenter.heroku.com/articles/classic-vs-cloud-native-buildpacks):

- **Classic buildpacks**: Configured via the `buildpacks` argument for specifying buildpack URLs or names.
- **CNBs**: Configured via [`project.toml`](https://devcenter.heroku.com/articles/managing-buildpacks#set-a-cloud-native-buildpack) in the source code. You can't use the `buildpacks` argument for CNBs.

Apps inherit their [generation](https://devcenter.heroku.com/articles/generations) from where they're deployed:

- Cedar-generation apps, which use classic buildpacks, are deployed in the [Common Runtime](https://devcenter.heroku.com/articles/dyno-runtime#common-runtime) or in [Cedar Private Spaces](https://devcenter.heroku.com/articles/private-spaces#additional-features-for-cedar-private-spaces).
- Fir-generation apps, which use CNBs, are deployed to [Fir Private Spaces](https://devcenter.heroku.com/articles/private-spaces#fir-private-spaces).

## Source Code Layout

The code contained in the source directory or tarball must follow the layout required by the [buildpack](https://devcenter.heroku.com/articles/managing-buildpacks#classic-buildpacks-references)
or the `Dockerfile` for [container builds](https://devcenter.heroku.com/articles/build-docker-images-heroku-yml).

### Building with Buildpacks

Building with buildpacks is the default build process.

For apps without a buildpack set, the app searches the [official Heroku buildpacks](https://devcenter.heroku.com/articles/officially-supported-buildpacks) until it detects a match and uses the buildpack to compile the app.

You can require a [`Procfile`](https://devcenter.heroku.com/articles/procfile) to successfully launch the app.
Some buildpacks provide a default web process, such as [`npm start` for Node.js](https://devcenter.heroku.com/articles/nodejs-support#default-web-process-type).
Other buildpacks can require a `Procfile`, like for a [pure Ruby app](https://devcenter.heroku.com/articles/ruby-support#ruby-applications-process-types).

### Building with Docker

To use container builds, set the parent `heroku_app` resource's `stack = "container"`

A [`heroku.yml` manifest](https://devcenter.heroku.com/articles/build-docker-images-heroku-yml#heroku-yml-overview)
file is required to declare which `Dockerfile` to build for each process. Be careful not to create conflicting configuration between `heroku.yml` and Terraform, such as add-ons or config vars.

### Building with Cloud Native Buildpacks

>[!NOTE]
>Fir-generation apps always use Cloud Native Buildpacks instead of classic buildpacks. 

You must specify the buildpack configuration in a `project.toml` file in the source code rather than the Terraform configuration. You can't use `buildpacks` argument to configure them. Attempting to do so results in an error during `terraform apply`.

Example `project.toml` for a Node.js app:

```toml
[build]
buildpacks = ["heroku/nodejs"]

[[build.env]]
name = "NODE_ENV"
value = "production"
```

For more information, see [Set a Cloud Native Buildpack](https://devcenter.heroku.com/articles/managing-buildpacks#set-a-cloud-native-buildpack).

## Source URLs
A `source.url` can point to any `https://` URL that responds to a `GET` with a tarball source code. When running `terraform apply`,
the source code is only fetched once for a successful build. Change the URL to force a new resource.

>[!NOTE]
>Source URLs are useful for building public, open-source source code, such as projects that publish releases on GitHub.
>
>They're not useful for private URLs that require credentials to access.

### GitHub URLs
GitHub provides [release](https://help.github.com/articles/creating-releases/) tarballs through URLs. Create a release
and then use the tag as a `source.url`, such as:

```
https://github.com/username/example/archive/v1.0.0.tar.gz
```

Using a branch or master `source.url` is possible, but be aware that tracking down exactly what commit was deployed
for a given `terraform apply` can be difficult. On the other hand, using stable release tags ensures repeatability
of the Terraform configuration.

### Example Usage with Source URL

#### Classic Buildpacks

```hcl-terraform
resource "heroku_app" "cedar_app" {
    name   = "my-cedar-app"
    region = "us"
}

resource "heroku_build" "cedar_build" {
  app_id     = heroku_app.cedar_app.id
  buildpacks = ["https://github.com/mars/create-react-app-buildpack"]

  source {
    # This app uses a community buildpack, set it in `buildpacks` above.
    url     = "https://github.com/mars/cra-example-app/archive/v2.1.1.tar.gz"
    version = "v2.1.1"
  }
}

resource "heroku_formation" "cedar_formation" {
  app_id     = heroku_app.cedar_app.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.cedar_build"]
}
```

#### Cloud Native Buildpacks

```hcl-terraform
resource "heroku_space" "fir_space" {
  name         = "my-fir-space"
  organization = "my-organization"
  region       = "virginia"
  generation   = "fir"
}

resource "heroku_app" "fir_app" {
  name   = "my-fir-app"
  region = heroku_space.fir_space.region
  space  = heroku_space.fir_space.id
  
  organization {
    name = "my-organization"
  }
}

resource "heroku_build" "fir_build" {
  app_id = heroku_app.fir_app.id
  # Note: Don't specify buildpacks for Fir apps
  # Buildpacks are configured via project.toml in the source code

  source {
    # Source must include project.toml for CNB configuration
    url     = "https://github.com/username/my-cnb-app/archive/v1.0.0.tar.gz"
    version = "v1.0.0"
  }
}

resource "heroku_formation" "fir_formation" {
  app_id     = heroku_app.fir_app.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.fir_build"]
}
```

## Local Source
A `source.path` can point to either:

* A tarball of source code
* A directory of source code
  * Use `src/appname` relative paths to subdirectories within the Terraform project repo (recommended).
  * Use `/opt/src/appname` absolute or `../appname` relative paths to external directories.
  * **Avoid ancestor paths that contain the Terraform configuration itself.**
    * Paths such as `../` [cause errors during apply](https://github.com/heroku/terraform-provider-heroku/issues/269)

When running `terraform apply`, if the contents (SHA256) of the source path changed since the last `apply`, then a new build starts.

>[!NOTE]
>The complete source must already be present at its `path` when Terraform runs, so either:
>  * Copy, clone, or check out the source to the `path` before Terraform runs, like [this issue's solution](https://github.com/heroku/terraform-provider-heroku/issues/321#issuecomment-926778363).
>  * Commit the source code into a subdirectory of the Terraform project repository, so that it's all cloned together.


### Example Usage with Local Source Directory

#### Classic Buildpacks

```hcl-terraform
resource "heroku_app" "cedar_app" {
    name   = "my-cedar-app"
    region = "us"
}

resource "heroku_build" "cedar_build" {
  app_id     = heroku_app.cedar_app.id
  buildpacks = ["heroku/nodejs"]

  source {
    # A local directory, changing its contents will
    # force a new build during `terraform apply`
    path = "src/my-cedar-app"
  }
}

resource "heroku_formation" "cedar_formation" {
  app_id     = heroku_app.cedar_app.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.cedar_build"]
}
```

#### Cloud Native Buildpacks

```hcl-terraform
resource "heroku_space" "fir_space" {
  name         = "my-fir-space"
  organization = "my-organization"
  region       = "virginia"
  generation   = "fir"
}

resource "heroku_app" "fir_app" {
  name   = "my-fir-app"
  region = heroku_space.fir_space.region
  space  = heroku_space.fir_space.id
  
  organization {
    name = "my-organization"
  }
}

resource "heroku_build" "fir_build" {
  app_id = heroku_app.fir_app.id
  # Note: Do not specify buildpacks for Fir apps

  source {
    # Local directory must contain project.toml
    # for Cloud Native Buildpack configuration
    path = "src/my-cnb-app"
  }
}

resource "heroku_formation" "fir_formation" {
  app_id     = heroku_app.fir_app.id
  type       = "web"
  quantity   = 1
  size       = "Standard-1x"
  depends_on = ["heroku_build.fir_build"]
}
```

## Argument Reference

The resource supports the following arguments:

* `app_id`: (Required) The Heroku app ID (don't use app name).
* `buildpacks`: (Optional) Buildpack GitHub URLs for the application. **Note:** Not supported for apps using Cloud Native Buildpacks, like Fir-generation apps. Use `project.toml` for configuration instead.
* `source`: (Required) A block that specifies the source code to build and release:
  * `checksum`: SHA256 hash of the tarball archive to verify its integrity, for example:
    `SHA256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`
  * `path`: (Required unless `source.url` is set) The local path to the source directory or tarball archive for the app.
  * `url`: (Required unless `source.path` is set) The `https` location of the source archive for the app.
  * `version`: Use to track what version of your source originated this build. If you are creating builds
    from git-versioned source code, for example, the commit hash, or release tag would be a good value to use for the
    version parameter.


## Attributes Reference

The following attributes are exported:

* `uuid`: The ID of the build
* `output_stream_url`: The URL that [streams the log output from the build](https://devcenter.heroku.com/articles/build-and-release-using-the-api#streaming-build-output).
* `release_id`: The Heroku app release created with a build's artifacts.
* `slug_id`: The Heroku slug created by a build. **Note**: Only for apps using classic buildpacks.
* `stack`: The name or ID of the [Heroku stack](https://devcenter.heroku.com/articles/stack).
* `status`: The status of a build. Possible values are `pending`, `successful` and `failed`.
* `user`: The Heroku account that created a build.
  * `email`: The email address of the user.
  * `id`: The ID of the user.

## Import
Import existing builds with a combination of the application name, a colon, and the build ID.

For example:
```
$ terraform import heroku_build.foobar bazbux:4f1db8ef-ed5c-4c42-a3d6-3c28262d5abc
```

* `foobar` is the **heroku_build** resource's name.
* `bazbux` is the Heroku app name or ID that the build belongs to.
* `:` separates the app identifier and the build identifier.
* `4f1db8ef…` is the build ID.
