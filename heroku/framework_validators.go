package heroku

import (
	"context"
	"fmt"
	"net"

	uuid "github.com/hashicorp/go-uuid"
	fwvalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// uuidStringValidator validates that a string attribute is a valid UUID,
// mirroring the SDKv2 validation.IsUUID ValidateFunc. The framework validators
// library does not ship a UUID validator, so this shared implementation is
// reused across the resources whose app/pipeline/owner identifiers must be
// UUIDs.
type uuidStringValidator struct{}

// uuidValidator returns the shared IsUUID validator.
func uuidValidator() fwvalidator.String { return uuidStringValidator{} }

func (v uuidStringValidator) Description(_ context.Context) string {
	return "value must be a valid UUID"
}

func (v uuidStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v uuidStringValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if _, err := uuid.ParseUUID(val); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid UUID",
			fmt.Sprintf("expected %q to be a valid UUID, got %s", req.Path, val),
		)
	}
}

// ociImageStringValidator validates that a string is either a valid UUID or a
// SHA256 digest, mirroring the SDKv2 validateOCIImage ValidateFunc on
// heroku_app_release.oci_image. It delegates to the shared validateOCIImage so
// the accepted formats stay in sync with the (still-used) SDKv2 helper.
type ociImageStringValidator struct{}

// ociImageValidator returns the shared OCI image identifier validator.
func ociImageValidator() fwvalidator.String { return ociImageStringValidator{} }

func (v ociImageStringValidator) Description(_ context.Context) string {
	return "value must be a UUID or SHA256 digest"
}

func (v ociImageStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ociImageStringValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, errs := validateOCIImage(req.ConfigValue.ValueString(), req.Path.String()); len(errs) > 0 {
		for _, err := range errs {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid OCI image identifier", err.Error())
		}
	}
}

// cidrNetworkStringValidator validates that a string is a valid CIDR network
// whose significant bits fall within [min, max], mirroring the SDKv2
// validation.IsCIDRNetwork ValidateFunc used on
// heroku_space_inbound_ruleset.rule.source (IsCIDRNetwork(0, 32)). The
// framework validators library does not ship a CIDR validator.
type cidrNetworkStringValidator struct {
	min int
	max int
}

// cidrNetworkValidator returns a validator enforcing a CIDR network with
// between min and max significant bits.
func cidrNetworkValidator(min, max int) fwvalidator.String {
	return cidrNetworkStringValidator{min: min, max: max}
}

func (v cidrNetworkStringValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be a valid CIDR network with between %d and %d significant bits", v.min, v.max)
}

func (v cidrNetworkStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrNetworkStringValidator) ValidateString(_ context.Context, req fwvalidator.StringRequest, resp *fwvalidator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()

	_, ipnet, err := net.ParseCIDR(val)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR network",
			fmt.Sprintf("expected %q to be a valid CIDR network, got %s: %v", req.Path, val, err),
		)
		return
	}

	if ipnet == nil || val != ipnet.String() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR network",
			fmt.Sprintf("expected %q to contain a valid network value, expected %v, got %s", req.Path, ipnet, val),
		)
		return
	}

	sigbits, _ := ipnet.Mask.Size()
	if sigbits < v.min || sigbits > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR network",
			fmt.Sprintf("expected %q to contain a network value with between %d and %d significant bits, got: %d", req.Path, v.min, v.max, sigbits),
		)
	}
}
