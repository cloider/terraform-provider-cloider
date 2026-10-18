package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// These run Terraform against fakeAPI: plans that come back empty after an
// apply, updates in place, and imports that read back what was applied.

func TestClusterLifecycle(t *testing.T) {
	server := newFakeAPI(t)
	config := func(paused bool) string {
		return fakeProviderConfig(server) + `
resource "cloider_cluster" "web" {
  name   = "web"
  paused = ` + map[bool]string{true: "true", false: "false"}[paused] + `
}
`
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_cluster.web", tfjsonpath.New("version"), knownvalue.StringExact("1.34")),
					statecheck.ExpectKnownValue("cloider_cluster.web", tfjsonpath.New("status"), knownvalue.StringExact("Ready")),
					statecheck.ExpectKnownValue("cloider_cluster.web", tfjsonpath.New("kube_config").AtMapKey("host"), knownvalue.StringExact("https://web.k8s.cloider.com")),
					statecheck.ExpectKnownValue("cloider_cluster.web", tfjsonpath.New("kube_config").AtMapKey("token"), knownvalue.StringExact("kube-token")),
				},
			},
			{
				Config: config(true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("cloider_cluster.web", plancheck.ResourceActionUpdate)},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_cluster.web", tfjsonpath.New("status"), knownvalue.StringExact("Paused")),
				},
			},
			{
				ResourceName:            "cloider_cluster.web",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestDatabaseLifecycle(t *testing.T) {
	server := newFakeAPI(t)
	config := func(body string) string {
		return fakeProviderConfig(server) + `
resource "cloider_database" "orders" {
  name   = "orders-db"
  engine = "postgresql"
` + body + `
}
`
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("user"), knownvalue.StringExact("orders_db_user")),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("version"), knownvalue.StringExact("18")),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("plan"), knownvalue.StringExact("hobby")),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("ssl"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("trusted_sources"), knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("*")})),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("port"), knownvalue.Int64Exact(5432)),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("password"), knownvalue.StringExact("s3cret")),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("ca_certificate"), knownvalue.StringExact("CA PEM")),
				},
			},
			{
				// A larger plan, a label and other sources change it in place
				Config: config(`
  plan            = "starter"
  label           = "orders"
  trusted_sources = ["203.0.113.0/24"]
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("cloider_database.orders", plancheck.ResourceActionUpdate)},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("plan"), knownvalue.StringExact("starter")),
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("label"), knownvalue.StringExact("orders")),
				},
			},
			{
				Config: config(`
  plan            = "starter"
  label           = "orders"
  trusted_sources = ["203.0.113.0/24"]
  paused          = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_database.orders", tfjsonpath.New("status"), knownvalue.StringExact("Paused")),
				},
			},
			{
				// A paused database can't be resized
				Config: config(`
  plan            = "basic"
  label           = "orders"
  trusted_sources = ["203.0.113.0/24"]
  paused          = true
`),
				ExpectError: regexp.MustCompile(`Can't resize a paused database`),
			},
			{
				// A smaller plan replaces it
				Config: config(`
  label           = "orders"
  trusted_sources = ["203.0.113.0/24"]
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("cloider_database.orders", plancheck.ResourceActionDestroyBeforeCreate)},
				},
			},
			{
				ResourceName:            "cloider_database.orders",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestDomainAndDnsRecordLifecycle(t *testing.T) {
	server := newFakeAPI(t)
	config := func(values string, ttl string) string {
		return fakeProviderConfig(server) + `
resource "cloider_domain" "example" {
  name = "example.com"
}

resource "cloider_dns_record" "www" {
  domain = cloider_domain.example.name
  type   = "A"
  name   = "www"
  values = ` + values + `
  ttl    = ` + ttl + `
}
`
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`["203.0.113.10"]`, "3600"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_domain.example", tfjsonpath.New("type"), knownvalue.StringExact("CustomDomain")),
					statecheck.ExpectKnownValue("cloider_dns_record.www", tfjsonpath.New("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("203.0.113.10")})),
				},
			},
			{
				Config: config(`["203.0.113.10", "203.0.113.11"]`, "300"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("cloider_dns_record.www", plancheck.ResourceActionUpdate)},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_dns_record.www", tfjsonpath.New("values"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue("cloider_dns_record.www", tfjsonpath.New("ttl"), knownvalue.Int64Exact(300)),
				},
			},
			{ResourceName: "cloider_dns_record.www", ImportState: true, ImportStateVerify: true},
			{ResourceName: "cloider_domain.example", ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestDomainAlreadyAddedIsRefused(t *testing.T) {
	server := newFakeAPI(t)
	claim := fakeProviderConfig(server) + `
resource "cloider_domain" "first" {
  name = "my-app.cloider.app"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: claim,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("cloider_domain.first", tfjsonpath.New("type"), knownvalue.StringExact("Subdomain")),
				},
			},
			{
				Config: claim + `
resource "cloider_domain" "again" {
  name = "my-app.cloider.app"
}
`,
				ExpectError: regexp.MustCompile(`Domain already added`),
			},
		},
	})
}
