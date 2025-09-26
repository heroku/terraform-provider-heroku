---
layout: "heroku"
page_title: "Heroku: heroku_telemetry_drain"
sidebar_current: "docs-heroku-resource-telemetry-drain"
description: |-
  Provides a Heroku Telemetry Drain resource. Use this resource to create a telemetry drain for Fir-generation apps and spaces.
---

# heroku\_telemetry\_drain

Provides a [Heroku Telemetry Drain](https://devcenter.heroku.com/articles/platform-api-reference#telemetry-drain) resource.
Telemetry drains forward OpenTelemetry traces, metrics, and logs from [Fir-generation](https://devcenter.heroku.com/articles/generations#fir) apps and spaces to your own consumer endpoint. 

Use this resource to create a [telemetry drain](https://devcenter.heroku.com/articles/heroku-telemetry) scoped to the app or space level.
You can create multiple telemetry drains per app or space.

## Generation Compatibility

Telemetry drains are **only supported for Fir-generation** apps and spaces. [Cedar-generation](https://devcenter.heroku.com/articles/generations#cedar) apps can use the [`heroku_drain`](./drain.html) resource for log forwarding instead.

## Signal Filtering

You can choose which signals to send in the Terraform configuration. Refer to the [Logs-Only Telemetry Drain](#logs-only-telemetry-drain) code snippet for an example.

## Example Usage

### App-Scoped Telemetry Drain

```hcl
resource "heroku_space" "fir_space" {
  name         = "my-fir-space"
  organization = "my-org"
  region       = "virginia"
  generation   = "fir"
}

resource "heroku_app" "fir_app" {
  name   = "my-fir-app"
  region = "virginia"
  space  = heroku_space.fir_space.name
  
  organization {
    name = "my-org"
  }
}

resource "heroku_telemetry_drain" "app_traces" {
  owner_id      = heroku_app.fir_app.id
  owner_type    = "app"
  endpoint      = "https://api.honeycomb.io/v1/traces"
  exporter_type = "otlphttp"
  signals       = ["traces", "metrics"]
  
  headers = {
    "x-honeycomb-team"    = var.honeycomb_api_key
    "x-honeycomb-dataset" = "my-service"
  }
}
```

### Space-Scoped Telemetry Drain

```hcl
resource "heroku_telemetry_drain" "space_observability" {
  owner_id      = heroku_space.fir_space.id
  owner_type    = "space"
  endpoint      = "otel-collector.example.com:4317"
  exporter_type = "otlp"
  signals       = ["traces", "metrics", "logs"]
  
  headers = {
    "Authorization" = "Bearer ${var.collector_token}"
  }
}
```

### Logs-Only Telemetry Drain

```hcl
resource "heroku_telemetry_drain" "app_logs" {
  owner_id      = heroku_app.fir_app.id
  owner_type    = "app"
  endpoint      = "https://logs.datadog.com/api/v2/logs"
  exporter_type = "otlphttp"
  signals       = ["logs"]
  
  headers = {
    "DD-API-KEY" = var.datadog_api_key
  }
}
```

## Argument Reference

The resource supports the following arguments:

* `owner_id`: (Required, ForceNew) The UUID of the app or space that owns this telemetry drain. You can't change it after creation.
* `owner_type`: (Required, ForceNew) The type of owner (`"app"` or `"space"`). You can't change it after creation.
* `endpoint`: (Required) The URI of your OpenTelemetry consumer endpoint.
* `exporter_type`: (Required) The transport type for your OpenTelemetry consumer. Must be either:
  * `"otlphttp"`: HTTP/HTTPS endpoints (example: `https://api.example.com/v1/traces`)
  * `"otlp"`: gRPC endpoints in `host:port` format (example: `collector.example.com:4317`)
* `signals`: (Required) An array of OpenTelemetry signals to send to the telemetry drain. Valid values are:
  * `"traces"`: The path of requests through your application.
  * `"metrics"`: The application and system metrics.
  * `"logs"`: The application and system logs.
* `headers`: (Required) The map of headers to send to your OpenTelemetry consumer for authentication or configuration. You must specify at least one header.

## Attributes Reference

The following attributes are exported:

* `id`: The UUID of the telemetry drain.
* `created_at`: The timestamp when the telemetry drain was created.
* `updated_at`: The timestamp when the telemetry drain was last updated.

## Endpoint Format Requirements

The `endpoint` format depends on the `exporter_type`:

* **otlphttp**: Full HTTP/HTTPS URL (example: `https://api.honeycomb.io/v1/traces`)
* **otlp**: Host and port only (example: `collector.example.com:4317`)

## Headers

The `headers` field supports custom key-value pairs for authentication and configuration:

* **Keys**: Must match the pattern `^[A-Za-z0-9\-_]{1,100}$` (alphanumeric, hyphens, underscores, max 100 chars)
* **Values**: Maximum 1000 characters each
* **Limit**: Maximum 20 header pairs per telemetry drain

Common header patterns:
* **API Keys**: `"Authorization" = "Bearer token"` or `"x-api-key" = "key"`
* **Content Types**: `"Content-Type" = "application/x-protobuf"`
* **Service Tags**: `"x-service" = "my-app"`, `"x-environment" = "production"`

## Validation

The provider performs generation-aware validation:

1. **Plan-time**: Schema validation for field types, required fields, and enum values
2. **Apply-time**: Generation compatibility check via Heroku API
   * Fetches app/space information to determine generation
   * Returns descriptive error if Cedar generation detected
   * Suggests using `heroku_drain` for Cedar apps

## Import

Import a telemetry drain with the drain `id`:

```
$ terraform import heroku_telemetry_drain.example 01234567-89ab-cdef-0123-456789abcdef
```
