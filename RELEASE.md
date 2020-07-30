## Release Flow

Since the migration to the [Terraform registry](https://registry.terraform.io/), this repository's maintainers now have
the ability to self-publish Terraform Heroku provider releases. This process leverages Github Actions
and [`goreleaser`](https://github.com/goreleaser/goreleaser) to build, sign, and upload provider binaries to a Github release.

The release flow is as follows:
1. Create a [new Github release](https://github.com/heroku/terraform-provider-heroku/releases/new).
    - For the 'Tag version' and 'Release title' fields, please enter a new & valid semantic version such as `v1.2.3`.
    - For the 'Describe this release' field, please follow the following format:
    ```markdown
    ## FEATURES:
    - Some text that describes the pull request (#123)

    ## IMPROVEMENTS:
    ...

    ## BUG FIXES:
    ...
    ```
1. Click 'Publish release' button.
    - Note: Draft releases will not trigger the release workflow or show up in the Terraform registry.
1. Github Actions will trigger the release workflow which can be
[viewed here](https://github.com/heroku/terraform-provider-heroku/actions?query=workflow%3Arelease).
After the workflow executes successfully, the Github release created in the prior step will
have the relevant assets available for consumption.
1. The new release will show up in https://registry.terraform.io/providers/heroku/heroku/latest for consumption
by terraform `0.13.X` users.
1. For terraform `0.12.X` users, the new release is available for consumption once it is present in
https://releases.hashicorp.com/terraform-provider-heroku/.

