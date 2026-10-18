package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloider": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck needs a Read & Write token for an account the acceptance tests
// may create and delete in, CLOIDER_TOKEN or the CLI's login.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CLOIDER_TOKEN") == "" && os.Getenv("CLOIDER_TOKEN_FILE") == "" {
		t.Log("no CLOIDER_TOKEN, using the cloider CLI's login")
	}
}

// TestProviderSchemas has the framework check every schema, which it otherwise
// only does when Terraform first asks for them: a default on an attribute that
// isn't computed, or a plan modifier on a data source attribute, fail here.
func TestProviderSchemas(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("%s: %s", d.Summary, d.Detail)
	}

	want := []string{"cloider_cluster", "cloider_database", "cloider_bucket", "cloider_bucket_key", "cloider_registry", "cloider_registry_token", "cloider_domain", "cloider_dns_record"}
	for _, name := range want {
		if _, ok := resp.ResourceSchemas[name]; !ok {
			t.Errorf("no %s resource", name)
		}
	}
	for _, name := range []string{"cloider_options", "cloider_bucket_regions"} {
		if _, ok := resp.DataSourceSchemas[name]; !ok {
			t.Errorf("no %s data source", name)
		}
	}
}
