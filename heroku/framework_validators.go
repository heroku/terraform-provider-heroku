package heroku

import (
	"context"
	"fmt"

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
