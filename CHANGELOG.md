## 0.2.0 (Unreleased)

FEATURES:

* r/heroku_app: Add `acm` field to enable Heroku Automated Certificate Management (ACM) [GH-38]
* r/heroku_team_collaborator: Add a new team collaborator resource to the provider ([#56](https://github.com/terraform-providers/terraform-provider-heroku/pull/56))
* d/heroku_space_peering_info: Add a new data resource for getting VPC peering information for a Heroku private space ([#57](https://github.com/terraform-providers/terraform-provider-heroku/pull/57))

IMPROVEMENTS:

* r/heroku_app: Wait until for new release after updating config vars [GH-35]

## 0.1.2 (January 04, 2018)

FEATURES:

* All resources now support `terraform import` ([#31](https://github.com/terraform-providers/terraform-provider-heroku/pull/31))

IMPROVEMENTS:

* r/heroku_app: Revert change ([#17](https://github.com/terraform-providers/terraform-provider-heroku/pull/17)) which deleted externally-created config vars ([#36](https://github.com/terraform-providers/terraform-provider-heroku/pull/36))
* r/heroku_app: Change import to use the app name as its ID if possible ([#34](https://github.com/terraform-providers/terraform-provider-heroku/pull/34))
* r/heroku_addon: Pass confirm option during addon creation ([#32](https://github.com/terraform-providers/terraform-provider-heroku/pull/32))
* r/heroku_space: Support trusted_ip_ranges ([#28](https://github.com/terraform-providers/terraform-provider-heroku/pull/28))

## 0.1.1 (November 07, 2017)

FEATURES:

* **New Resource:** `r/heroku_addon_attachment` ([#19](https://github.com/terraform-providers/terraform-provider-heroku/issues/19))

IMPROVEMENTS:

* r/heroku_app: Protect against panic ([#11](https://github.com/terraform-providers/terraform-provider-heroku/issues/11))
* r/heroku_app: always read all config vars ([#17](https://github.com/terraform-providers/terraform-provider-heroku/issues/17))
* r/heroku_app: Handle updating an app's stack ([#16](https://github.com/terraform-providers/terraform-provider-heroku/issues/16))
* r/heroku_app: Adding an Exists method to check for if an App exists ([#20](https://github.com/terraform-providers/terraform-provider-heroku/issues/20))
* r/heroku_app: Making the `config_vars` field Optional + Computed ([#22](https://github.com/terraform-providers/terraform-provider-heroku/issues/22)] [[#23](https://github.com/terraform-providers/terraform-provider-heroku/issues/23))
* r/heroku_addon: Adding an Exists method to check for if an Addon exists ([#21](https://github.com/terraform-providers/terraform-provider-heroku/issues/21))

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
