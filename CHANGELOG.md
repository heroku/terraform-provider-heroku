# This file is now deprecated. Please visit [Releases](https://github.com/heroku/terraform-provider-heroku/releases) for more information.

## 2.5.0 (July 03, 2020)

FEATURES:
* New data source: `heroku_pipeline` Get information on a Heroku Pipeline ([#268](https://github.com/heroku/terraform-provider-heroku/pull/268))

IMPROVEMENTS:
* Clarify usage constraints of Build source path ([#270](https://github.com/heroku/terraform-provider-heroku/pull/270))
* Upgrade acceptance tests to Go 1.14 ([#271](https://github.com/heroku/terraform-provider-heroku/pull/271))

## 2.4.1 (May 20, 2020)

IMPROVEMENTS:
* Upgrade to Terraform Plugin SDK v1.12.0 ([#266](https://github.com/heroku/terraform-provider-heroku/pull/266))

BUG FIXES:
* Importing a `heroku_pipeline` by its name now sets its ID correctly ([#266](https://github.com/heroku/terraform-provider-heroku/pull/266))

## 2.4.0 (April 22, 2020)

FEATURES:
* **Resource `heroku_pipeline`** now supports setting `owner` user or team, including defaulting to the current API key's user ID ([#259](https://github.com/heroku/terraform-provider-heroku/pull/259))

IMPROVEMENTS:
* **Terraform Provider acceptance tests** now run on pull requests, pushes to master, and nightly using [GitHub Actions](https://github.com/heroku/terraform-provider-heroku/actions) configured for the [CI workflows](https://github.com/heroku/terraform-provider-heroku/tree/master/.github/workflows)

BUG FIXES:
* **Resource `heroku_app`**
  * now imports `buildpacks` and other attributes, consistent with create & read ([#257](https://github.com/heroku/terraform-provider-heroku/pull/257))
  * now reads `organization.locked`, consistent with create ([#257](https://github.com/heroku/terraform-provider-heroku/pull/257))
  * drops the non-standard `uuid` attribute ([#257](https://github.com/heroku/terraform-provider-heroku/pull/257))
* **Data source `heroku_app`** now returns `id` attribute, consistent with resource `heroku_app` ([#255](https://github.com/heroku/terraform-provider-heroku/pull/255))


## 2.3.0 (March 30, 2020)

FEATURES:
* New resource: `heroku_pipeline_config_var` ([#256](https://github.com/heroku/terraform-provider-heroku/pull/256))

IMPROVEMENTS:
* Upgrade `heroku-go` to `v5.2.0` ([#256](https://github.com/heroku/terraform-provider-heroku/pull/256))

BUG FIXES:
* Properly set `heroku_app.acm` when this attribute is not defined ([#256](https://github.com/heroku/terraform-provider-heroku/pull/256))
* Remove quoted interpolation-only expressions in docs ([#245](https://github.com/heroku/terraform-provider-heroku/pull/245))

## 2.2.2 (February 20, 2020)

IMPROVEMENTS
* Upgrade to Terraform Plugin SDK v1.7.0 ([#248](https://github.com/heroku/terraform-provider-heroku/pull/248))

## 2.2.1 (October 03, 2019)

IMPROVEMENTS:
* Migrate to Terraform Plugin SDK ([#240](https://github.com/heroku/terraform-provider-heroku/pull/240))

## 2.2.0 (September 19, 2019)

FEATURES:
* `heroku_app_webhook` - Ability to manage App Webhooks ([#239](https://github.com/heroku/terraform-provider-heroku/pull/239))

IMPROVEMENTS:
* Update vendored Terraform to v0.12.8 ([#238](https://github.com/heroku/terraform-provider-heroku/pull/238))

## 2.1.2 (August 09, 2019)

IMPROVEMENTS:
* Update vendored Terraform to v0.12.6 ([#234](https://github.com/heroku/terraform-provider-heroku/pull/234))
* Stop creating `cedar-14` apps during tests ([#232](https://github.com/heroku/terraform-provider-heroku/pull/232))
* Standardize UUID usage on `google/uuid` ([#228](https://github.com/heroku/terraform-provider-heroku/pull/228))

## 2.1.1 (August 07, 2019)

BUG FIXES:
* Rework `heroku_addon.config` migration ([#230](https://github.com/heroku/terraform-provider-heroku/pull/230))
* Fix `heroku_formation` segfault when app does not exist ([#229](https://github.com/heroku/terraform-provider-heroku/pull/229))
* Docs correction ([#225](https://github.com/heroku/terraform-provider-heroku/pull/225)) & clarification ([#224](https://github.com/heroku/terraform-provider-heroku/pull/224))


## 2.1.0 (July 24, 2019)

FEATURES:
* `heroku_addon` - Ability to set addon name ([#210](https://github.com/heroku/terraform-provider-heroku/pull/210))

BUG FIXES:
* Add migration for `heroku_addon` to fix dirty plan after `heroku_addon.config` attribute changed from `TypeList` of `TypeSet`
 to `TypeSet` ([#217](https://github.com/heroku/terraform-provider-heroku/pull/217))

## 2.0.3 (July 12, 2019)

BUG FIXES:
* Restore compatibility with Terraform 0.12 [#220](https://github.com/heroku/terraform-provider-heroku/pull/220)

## 2.0.2 (July 11, 2019)

**This release is broken for Terraform 0.12.**

IMPROVEMENTS:
* Upgrade heroku-go client to stable Go Module support `v5` & to provide more Heroku Platform API features [#211](https://github.com/heroku/terraform-provider-heroku/pull/211)

BUG FIXES:
* Revise docs for Terraform v0.12 map attribute-assignment syntax [#212](https://github.com/heroku/terraform-provider-heroku/pull/212) & [#216](https://github.com/heroku/terraform-provider-heroku/pull/216)
* Revise docs to correct `import` example [#215](https://github.com/heroku/terraform-provider-heroku/pull/215)

## 2.0.1 (June 20, 2019)

BUG FIXES:
* Fix unhandled errors [#193](https://github.com/heroku/terraform-provider-heroku/pull/193)
* Prevent leaking `heroku_app` `sensitive_config_vars` through `all_config_vars` [#206](https://github.com/heroku/terraform-provider-heroku/pull/206)

## 2.0.0 (June 03, 2019)

FEATURES:
* **Terraform 0.12** compatibility [#179](https://github.com/heroku/terraform-provider-heroku/pull/179)

IMPROVEMENTS:
* Documentation fix-ups [#189](https://github.com/heroku/terraform-provider-heroku/pull/189) & [#192](https://github.com/heroku/terraform-provider-heroku/pull/192)

## 1.9.0 (April 01, 2019)

FEATURES:
* **New Data Source:** `heroku_team` [#188](https://github.com/heroku/terraform-provider-heroku/pull/188)
* **New Resource:** `heroku_config` (for defining config vars to be used in other resources) [#183](https://github.com/heroku/terraform-provider-heroku/pull/183)
* **New Resource:** `heroku_app_config_association` (for setting, updating, and deleting config vars on apps) [#183](https://github.com/heroku/terraform-provider-heroku/pull/183)

IMPROVEMENTS:
* Clarify usage of Heroku Teams in the docs [#187](https://github.com/heroku/terraform-provider-heroku/pull/187)

BUG FIXES:
* Fix tests using SSL Endpoint DNS target [#191](https://github.com/heroku/terraform-provider-heroku/pull/191)

## 1.8.0 (February 27, 2019)

FEATURES:
* Switch to Go Modules (prep for Terraform 0.12) [#177](https://github.com/heroku/terraform-provider-heroku/pull/177)

IMPROVEMENTS:
* Clarifying, expanding, and cross-referencing the Provider docs [#175](https://github.com/heroku/terraform-provider-heroku/pull/175)

BUG FIXES:
* Fix so `heroku_build` source path can be current `.` or a parent `..` directory [#181](https://github.com/heroku/terraform-provider-heroku/pull/181)

## 1.7.4 (February 01, 2019)

IMPROVEMENTS:
* Upgrade heroku-go client to support more Heroku Platform API features [#169](https://github.com/heroku/terraform-provider-heroku/pull/169)
* Add `cidr` & `data_cidr` to `heroku_space` resource & data source [#167](https://github.com/heroku/terraform-provider-heroku/pull/167)

## 1.7.3 (January 22, 2019)

IMPROVEMENTS:
* `heroku_app` - New attribute `sensitive_config_vars` to help with sensitive heroku app config vars [#163](https://github.com/heroku/terraform-provider-heroku/pull/163)

## 1.7.2 (January 08, 2019)

IMPROVEMENTS:
* Identify Terraform API requests via User-Agent header [#161](https://github.com/heroku/terraform-provider-heroku/pull/161)

## 1.7.1 (December 18, 2018)

BUG FIXES:
* Add missing features/fixes to changelog [#157](https://github.com/heroku/terraform-provider-heroku/pull/157)
* Build resource doc fixups [#156](https://github.com/heroku/terraform-provider-heroku/pull/156)

## 1.7.0 (December 14, 2018)

FEATURES:
* **New Resource:** `heroku_build` (for deploying source code to Heroku) [#149](https://github.com/heroku/terraform-provider-heroku/pull/149)

IMPROVEMENTS:
* Retry with backoff when rate-limited [#135](https://github.com/heroku/terraform-provider-heroku/pull/135)
* Configurable delays to provider to help alleviate issues with Heroku backend eventual consistency with regards to app, spaces, and domain creation [#142](https://github.com/heroku/terraform-provider-heroku/pull/142)

BUG FIXES:
* `heroku_app_feature` - Typos in Documentation & Test Fixes [#143](https://github.com/heroku/terraform-provider-heroku/pull/143)
* Fix bad formatting in docs [#147](https://github.com/heroku/terraform-provider-heroku/pull/147)
* Fix panic condition in parseCompositeID [#148](https://github.com/heroku/terraform-provider-heroku/pull/148)
* Terraform sometimes creates a Heroku Cert, then gets a conflict error [#37](https://github.com/heroku/terraform-provider-heroku/issues/37)
* Add Exponential Backoff When adding Heroku Domain [#71](https://github.com/heroku/terraform-provider-heroku/issues/71)
* heroku_space_inbound ruleset can't be applied because heroku_space isn't actually ready [#116](https://github.com/heroku/terraform-provider-heroku/issues/116)

## 1.6.0 (November 13, 2018)

FEATURES:
* **New Resource:** `heroku_account_feature` (for managing account features) [#134](https://github.com/heroku/terraform-provider-heroku/pull/134)
* **New Data Resource:** `heroku_addon` (Get information on a Heroku Addon) [#130](https://github.com/heroku/terraform-provider-heroku/pull/130)

IMPROVEMENTS:
* `heroku_cert` - Set private_key parameter to sensitive [#133](https://github.com/heroku/terraform-provider-heroku/pull/133)
* `heroku_slug` - Add test for heroku slug with a private space app [#138](https://github.com/heroku/terraform-provider-heroku/pull/138)
* `heroku_slug` - Fetch slug archives via HTTPS & allow users to specify a `file_url` attribute [#139](https://github.com/heroku/terraform-provider-heroku/pull/139)

BUG FIXES:
* `heroku_formation` - Fix formation.html.markdown [#136](https://github.com/heroku/terraform-provider-heroku/pull/136)
* `heroku_domain` - Fix heroku domain test after a [recent change](https://devcenter.heroku.com/changelog-items/1488) randomly generates a DNS target [#137](https://github.com/heroku/terraform-provider-heroku/pull/137)


## 1.5.0 (October 15, 2018)

FEATURES:
* Support for ~/.netrc authentication [#113](https://github.com/heroku/terraform-provider-heroku/pull/113)

IMPROVEMENTS:
* `heroku_app` - Now exports the UUID Heroku assigns the app as `uuid` [#127](https://github.com/heroku/terraform-provider-heroku/pull/127)
* `heroku_slug` - Slug doc corrections & formatting [#125](https://github.com/heroku/terraform-provider-heroku/pull/125)
* Fixes code snippet in the README.md [#129](https://github.com/heroku/terraform-provider-heroku/pull/129)

## 1.4.0 (September 11, 2018)

FEATURES:
* **New Resource:** `heroku_slug` - Provides the ability to create & upload a slug (archive of executable code) to an app [#119](https://github.com/heroku/terraform-provider-heroku/pull/119)
* **New Resource:** `heroku_team_member` - Ability to manage members of a Heroku team [#121](https://github.com/heroku/terraform-provider-heroku/pull/121)

IMPROVEMENTS:
* `heroku_app` - Remove notes about private beta for internal apps and VPN connections [#115](https://github.com/terraform-providers/terraform-provider-heroku/pull/115)
* `heroku_space_vpn_connection` - Remove notes about private beta for internal apps and VPN connections [#115](https://github.com/terraform-providers/terraform-provider-heroku/pull/115)

BUG FIXES:
* `data_source_heroku_space_peering_info` - Fix missing vpc_cidr attribute [#114](https://github.com/terraform-providers/terraform-provider-heroku/pull/114)
* logging - Case insensitive logging check for TF_LOG [#120](https://github.com/terraform-providers/terraform-provider-heroku/pull/120)

## 1.3.0 (August 16, 2018)

FEATURES:
* **New Resource:** `heroku_space_app_access` (for managing space access) [#83](https://github.com/terraform-providers/terraform-provider-heroku/pull/83)
* **New Resource:** `heroku_space_vpn_connection` (to establish VPN connections) [#104](https://github.com/terraform-providers/terraform-provider-heroku/pull/104)

IMPROVEMENTS:
* Replace custom validators with validators provided by terraform [#107](https://github.com/terraform-providers/terraform-provider-heroku/pull/107)
* Various test improvements and fixes [#105](https://github.com/terraform-providers/terraform-provider-heroku/pull/105)
* Update heroku dep to point at heroku repo [#106](https://github.com/terraform-providers/terraform-provider-heroku/pull/106)

BUG FIXES:
* Set vpc_peering_connection_id attribute to the correct value [#110](https://github.com/terraform-providers/terraform-provider-heroku/pull/110)

## 1.2.0 (July 21, 2018)

IMPROVEMENTS:

* Update the `heroku-go` client to the latest version. [#97](https://github.com/terraform-providers/terraform-provider-heroku/pull/97)
* Migrate to dep for managing `vendor/` and update packages in `vendor/` [#99](https://github.com/terraform-providers/terraform-provider-heroku/pull/99)

FEATURES:
* Add `internal_routing` option to `heroku_app` [#100](https://github.com/terraform-providers/terraform-provider-heroku/pull/100)

## 1.1.1 (July 17, 2018)

BUG FIXES:

* r/heroku_addon: Add migration to store the `uuid` for the resource ID instead of `name` ([#84](https://github.com/terraform-providers/terraform-provider-heroku/pull/84))
* r/heroku_addon_attachment: Add migration to store the `uuid` of the addon for `addon_id` ([#84](https://github.com/terraform-providers/terraform-provider-heroku/pull/84))

## 1.1.0 (July 13, 2018)

FEATURES:

* r/heroku_space_inbound_ruleset: Add a new resource for managing [inbound IP rulesets](https://devcenter.heroku.com/articles/platform-api-reference#inbound-ruleset) for Heroku Private Spaces ([#91](https://github.com/terraform-providers/terraform-provider-heroku/pull/91))

IMPROVEMENTS:

* HTTP headers are now shown when `TF_LOG` is set to `DEBUG` or `TRACE` ([#89](https://github.com/terraform-providers/terraform-provider-heroku/pull/89))

## 1.0.2 (July 10, 2018)

BUG FIXES:

* r/heroku_formation: Add support for Free/Hobby Dyno Types ([#80](https://github.com/terraform-providers/terraform-provider-heroku/pull/80))
* r/heroku_space: Fixed interface conversion panic applying changed trusted ips ([#88](https://github.com/terraform-providers/terraform-provider-heroku/pull/88))

## 1.0.1 (June 27, 2018)

BUG FIXES:

* r/heroku_space: Fix bug [#75](https://github.com/terraform-providers/terraform-provider-heroku/issues/75) in [#76](https://github.com/terraform-providers/terraform-provider-heroku/pull/76) that caused Terraform plans to destroy existing Heroku Private Spaces

## 1.0.0 (June 19, 2018)

FEATURES:

* r/heroku_app: Add `acm` field to enable Heroku Automated Certificate Management (ACM) ([#38](https://github.com/terraform-providers/terraform-provider-heroku/issues/38))
* r/heroku_team_collaborator: Add a new team collaborator resource ([#56](https://github.com/terraform-providers/terraform-provider-heroku/pull/56))
* d/heroku_space_peering_info: Add a new data resource for getting VPC peering information for a Heroku private space ([#57](https://github.com/terraform-providers/terraform-provider-heroku/pull/57))
* r/heroku_app_release: Add a new app release resource ([#62](https://github.com/terraform-providers/terraform-provider-heroku/pull/62))
* r/heroku_formation: Add a new formation resource ([#62](https://github.com/terraform-providers/terraform-provider-heroku/pull/62))
* r/heroku_space_peering_request_accepter: Add a new space peering request accepter resource ([#58](https://github.com/terraform-providers/terraform-provider-heroku/pull/58))

IMPROVEMENTS:

* r/heroku_app: Wait until for new release after updating config vars ([#35](https://github.com/terraform-providers/terraform-provider-heroku/issues/35))
* r/heroku_addon_attachment: Fix Attachment Ids
* r/heroku_space: Add space attributes for outbound IPs and shield

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
