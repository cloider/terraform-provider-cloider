package provider

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// trustedSourceValidator takes what the CLI's --trusted-source takes, with ids in
// place of names: '*', an IPv4 CIDR, cluster:<id> or app:<id>.
type trustedSourceValidator struct{}

func (trustedSourceValidator) Description(context.Context) string {
	return "'*', an IPv4 CIDR, cluster:<id> or app:<id>"
}

func (v trustedSourceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (trustedSourceValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "*" {
		return
	}
	if kind, id, ok := strings.Cut(value, ":"); ok && (kind == "cluster" || kind == "app") {
		if id == "" {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid trusted source", fmt.Sprintf("%q names no %s, use %s:<id>.", value, kind, kind))
		}
		return
	}

	ip, network, err := net.ParseCIDR(value)
	if err != nil || ip.To4() == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid trusted source", fmt.Sprintf("%q is not '*', an IPv4 CIDR like 203.0.113.0/24, cluster:<id> or app:<id>. A single address is <ip>/32.", value))
		return
	}
	// The api stores the network, so 10.0.0.7/24 would read back as 10.0.0.0/24
	if network.String() != value {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid trusted source", fmt.Sprintf("Write %q as %s, the network it allows.", value, network.String()))
	}
}

// anywhereAloneValidator keeps '*' on its own, as the dashboard does: allowing
// anywhere makes every other source moot.
type anywhereAloneValidator struct{}

func (anywhereAloneValidator) Description(context.Context) string {
	return "'*' can't be listed with other sources"
}

func (v anywhereAloneValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (anywhereAloneValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	values, diags := stringsOf(ctx, req.ConfigValue)
	if diags.HasError() {
		return
	}
	if len(values) > 1 && slices.Contains(values, "*") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid trusted sources", "'*' allows anywhere, so it can't be listed with other sources.")
	}
}
