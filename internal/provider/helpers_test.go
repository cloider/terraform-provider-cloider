package provider

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKubeConfigValue(t *testing.T) {
	ca := base64.StdEncoding.EncodeToString([]byte("CA PEM"))
	cert := base64.StdEncoding.EncodeToString([]byte("CERT PEM"))
	key := base64.StdEncoding.EncodeToString([]byte("KEY PEM"))
	raw := `apiVersion: v1
kind: Config
current-context: web
clusters:
- name: other
  cluster:
    server: https://other.example
- name: web
  cluster:
    server: https://web.k8s.cloider.com
    certificate-authority-data: ` + ca + `
contexts:
- name: web
  context:
    cluster: web
    user: admin
users:
- name: admin
  user:
    client-certificate-data: ` + cert + `
    client-key-data: ` + key + `
`

	attrs := kubeConfigValue(raw).Attributes()
	for name, want := range map[string]string{
		"raw":                    raw,
		"host":                   "https://web.k8s.cloider.com",
		"cluster_ca_certificate": "CA PEM",
		"client_certificate":     "CERT PEM",
		"client_key":             "KEY PEM",
	} {
		if got := attrs[name].(types.String).ValueString(); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if !attrs["token"].IsNull() {
		t.Errorf("token = %v, want null", attrs["token"])
	}

	if !kubeConfigValue("").IsNull() {
		t.Error("no kubeconfig yet should be null")
	}
	if got := kubeConfigValue("not: [yaml").Attributes()["raw"].(types.String).ValueString(); got != "not: [yaml" {
		t.Errorf("a kubeconfig that doesn't parse should still give raw, got %q", got)
	}
}

func TestTrustedSourceValidator(t *testing.T) {
	for value, valid := range map[string]bool{
		"*":              true,
		"203.0.113.0/24": true,
		"203.0.113.7/32": true,
		"cluster:x8f2k":  true,
		"app:p0q9z":      true,
		"203.0.113.7":    false,
		"203.0.113.7/24": false,
		"2001:db8::/32":  false,
		"cluster:":       false,
		"anywhere":       false,
	} {
		resp := &validator.StringResponse{}
		trustedSourceValidator{}.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("trusted_sources"), ConfigValue: types.StringValue(value)}, resp)
		if resp.Diagnostics.HasError() == valid {
			t.Errorf("%q: valid = %v, diagnostics %v", value, !resp.Diagnostics.HasError(), resp.Diagnostics)
		}
	}
}

func TestTrustedSourceStringRoundTrips(t *testing.T) {
	for _, s := range []client.TrustedSource{
		{Name: "Allow Everything", Code: "*", Type: "Anywhere"},
		{Name: "web", Code: "x8f2k", Type: "Cluster"},
		{Name: "api", Code: "p0q9z", Type: "App"},
		{Name: "203.0.113.0/24", Code: "203.0.113.0/24", Type: "Custom"},
		{Name: "My IP (198.51.100.4/32)", Code: "198.51.100.4/32", Type: "MyIP"},
	} {
		value := trustedSourceString(s)
		resp := &validator.StringResponse{}
		trustedSourceValidator{}.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("trusted_sources"), ConfigValue: types.StringValue(value)}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%+v reads back as %q, which the attribute doesn't take", s, value)
		}
	}
}

func TestPlanIndex(t *testing.T) {
	if planIndex("hobby") != 0 || planIndex("Pro") != len(databasePlans)-1 || planIndex("huge") != -1 {
		t.Fatal("planIndex should find plans case insensitively, smallest first")
	}
	if planIndex("starter") >= planIndex("standard") {
		t.Fatal("starter is smaller than standard")
	}
}
