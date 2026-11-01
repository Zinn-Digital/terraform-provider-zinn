package provider

import (
	"fmt"

	"github.com/Zinn-Digital/terraform-provider-zinn/internal/zinnapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func clientFrom(data any, diags *diag.Diagnostics) *zinnapi.Client {
	if data == nil {
		return nil // provider not configured yet (validation pass)
	}
	c, ok := data.(*zinnapi.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected *zinnapi.Client, got %T", data))
		return nil
	}
	return c
}

func needsKey(c *zinnapi.Client, diags *diag.Diagnostics) bool {
	if c.APIKey == "" {
		diags.AddError("API key required", "Set api_key in the provider block or ZINN_API_KEY in the environment. "+
			"Create one on the dashboard's API keys page.")
		return false
	}
	return true
}

func strPtr(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

func int64Ptr(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

func boolPtr(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}
