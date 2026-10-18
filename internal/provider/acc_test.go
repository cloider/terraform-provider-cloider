package provider

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/cloider/terraform-provider-cloider/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Acceptance tests create real resources in the account of CLOIDER_TOKEN (or the
// CLI's login) and only run with TF_ACC=1. They run one after another and each
// makes at most one of a kind, which a free account has room for:
//
//	TF_ACC=1 go test ./internal/provider -run TestAcc -timeout 60m

func testAccName() string {
	return "tfacc-" + strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
}

func testAccClient(t *testing.T) *client.Client {
	t.Helper()
	session, err := client.Resolve(client.Settings{})
	if err != nil {
		t.Fatal(err)
	}
	return client.New(session.APIURL, session.Token, "terraform-provider-cloider/acctest", os.Getenv("CLOIDER_INSECURE") == "1")
}

// testAccCheckDestroyed checks the api has none of the state's resources left.
func testAccCheckDestroyed(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := testAccClient(t)
		ctx := context.Background()
		for _, rs := range s.RootModule().Resources {
			id := rs.Primary.ID
			var err error
			gone := false
			switch rs.Type {
			case "cloider_cluster":
				var clusters []client.Cluster
				err = c.Get(ctx, "/clusters", nil, &clusters)
				gone = !slices.ContainsFunc(clusters, func(x client.Cluster) bool { return x.HashedId == id })
			case "cloider_database", "cloider_bucket", "cloider_registry":
				path := map[string]string{"cloider_database": "/databases/", "cloider_bucket": "/buckets/", "cloider_registry": "/registry/projects/"}[rs.Type]
				var status string
				status, err = c.GetText(ctx, path+id+"/status", nil)
				gone = client.IsNotFound(err) || strings.TrimSpace(status) == "Terminated"
			case "cloider_bucket_key":
				var keys []client.BucketKey
				err = c.Get(ctx, "/bucket-keys", nil, &keys)
				gone = !slices.ContainsFunc(keys, func(x client.BucketKey) bool { return x.HashedId == id })
			case "cloider_registry_token":
				var tokens []client.RegistryToken
				err = c.Get(ctx, "/registry/tokens", nil, &tokens)
				gone = !slices.ContainsFunc(tokens, func(x client.RegistryToken) bool { return x.HashedId == id })
			case "cloider_domain":
				err = c.Get(ctx, "/domains/"+id, nil, nil)
				gone = client.IsNotFound(err)
			case "cloider_dns_record":
				err = c.Get(ctx, "/dns-records/"+id, nil, nil)
				gone = client.IsNotFound(err)
			default:
				continue
			}
			if !gone {
				if err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("can't check %s %s is gone: %w", rs.Type, id, err)
				}
				return fmt.Errorf("%s %s still exists", rs.Type, id)
			}
		}
		return nil
	}
}

func TestAccOptionsAndRegions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
data "cloider_options" "all" {}
data "cloider_bucket_regions" "all" {}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.cloider_options.all", "cluster_versions.0"),
				resource.TestCheckResourceAttrSet("data.cloider_options.all", "database_engines.0.engine"),
				resource.TestCheckResourceAttr("data.cloider_options.all", "database_plans.0", "hobby"),
				resource.TestCheckResourceAttrSet("data.cloider_bucket_regions.all", "regions.0.code"),
			),
		}},
	})
}

func TestAccBucketAndKey(t *testing.T) {
	name := testAccName()
	config := func(public, versioning bool) string {
		return fmt.Sprintf(`
resource "cloider_bucket" "assets" {
  name       = %q
  public     = %t
  versioning = %t
}

resource "cloider_bucket_key" "deploy" {
  label   = "tfacc deploy"
  buckets = [cloider_bucket.assets.id]
}
`, name, public, versioning)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: config(false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_bucket.assets", "status", "Ready"),
					resource.TestCheckResourceAttrSet("cloider_bucket.assets", "endpoint"),
					resource.TestCheckResourceAttrSet("cloider_bucket_key.deploy", "access_key"),
					resource.TestCheckResourceAttrSet("cloider_bucket_key.deploy", "secret_key"),
				),
			},
			{
				Config: config(true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_bucket.assets", "public", "true"),
					resource.TestCheckResourceAttr("cloider_bucket.assets", "versioning", "true"),
				),
			},
			{
				// Versioning that was on can only be suspended, which reads back as off
				Config: config(true, false),
				Check:  resource.TestCheckResourceAttr("cloider_bucket.assets", "versioning", "false"),
			},
			{ResourceName: "cloider_bucket.assets", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"timeouts"}},
			{ResourceName: "cloider_bucket_key.deploy", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"secret_key"}},
		},
	})
}

func TestAccRegistryAndToken(t *testing.T) {
	name := testAccName()
	config := func(limit int, immutable bool) string {
		return fmt.Sprintf(`
resource "cloider_registry" "images" {
  name             = %q
  storage_limit_gi = %d
  immutable_tags   = %t
}

resource "cloider_registry_token" "ci" {
  label           = "tfacc ci"
  projects        = [cloider_registry.images.id]
  expires_in_days = 30
}
`, name, limit, immutable)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: config(5, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_registry.images", "status", "Ready"),
					resource.TestCheckResourceAttrSet("cloider_registry.images", "url"),
					resource.TestCheckResourceAttrSet("cloider_registry_token.ci", "username"),
					resource.TestCheckResourceAttrSet("cloider_registry_token.ci", "secret"),
					resource.TestCheckResourceAttrSet("cloider_registry_token.ci", "expires_at"),
				),
			},
			{
				Config: config(10, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_registry.images", "storage_limit_gi", "10"),
					resource.TestCheckResourceAttr("cloider_registry.images", "immutable_tags", "true"),
				),
			},
			{ResourceName: "cloider_registry.images", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"timeouts"}},
			{ResourceName: "cloider_registry_token.ci", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"secret"}},
		},
	})
}

func TestAccDomainAndDnsRecord(t *testing.T) {
	domain := testAccName() + ".example.com"
	config := func(ttl int) string {
		return fmt.Sprintf(`
resource "cloider_domain" "example" {
  name = %q
}

resource "cloider_dns_record" "www" {
  domain = cloider_domain.example.name
  type   = "A"
  name   = "www"
  values = ["203.0.113.10"]
  ttl    = %d
}

resource "cloider_dns_record" "spf" {
  domain = cloider_domain.example.name
  type   = "TXT"
  values = ["v=spf1 -all"]
  ttl    = 3600
}
`, domain, ttl)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: config(3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_domain.example", "type", "CustomDomain"),
					resource.TestCheckResourceAttr("cloider_dns_record.spf", "name", ""),
				),
			},
			{
				Config: config(300),
				Check:  resource.TestCheckResourceAttr("cloider_dns_record.www", "ttl", "300"),
			},
			{ResourceName: "cloider_domain.example", ImportState: true, ImportStateVerify: true},
			{ResourceName: "cloider_dns_record.www", ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestAccCluster(t *testing.T) {
	name := testAccName()
	config := func(paused bool) string {
		return fmt.Sprintf(`
resource "cloider_cluster" "web" {
  name   = %q
  paused = %t
}
`, name, paused)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: config(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_cluster.web", "status", "Ready"),
					resource.TestCheckResourceAttrSet("cloider_cluster.web", "kube_config.host"),
				),
			},
			{
				Config: config(true),
				Check:  resource.TestCheckResourceAttr("cloider_cluster.web", "status", "Paused"),
			},
			{ResourceName: "cloider_cluster.web", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"timeouts"}},
		},
	})
}

func TestAccDatabase(t *testing.T) {
	name := testAccName()
	config := func(label string) string {
		return fmt.Sprintf(`
resource "cloider_database" "cache" {
  name            = %q
  engine          = "redis"
  label           = %q
  trusted_sources = ["203.0.113.0/24"]
}
`, name, label)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestroyed(t),
		Steps: []resource.TestStep{
			{
				Config: config(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloider_database.cache", "status", "Ready"),
					resource.TestCheckResourceAttr("cloider_database.cache", "ssl", "false"),
					resource.TestCheckResourceAttrSet("cloider_database.cache", "host"),
					resource.TestCheckResourceAttrSet("cloider_database.cache", "password"),
				),
			},
			{
				Config: config(name + "-renamed"),
				Check:  resource.TestCheckResourceAttr("cloider_database.cache", "label", name+"-renamed"),
			},
			{ResourceName: "cloider_database.cache", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"timeouts"}},
		},
	})
}
