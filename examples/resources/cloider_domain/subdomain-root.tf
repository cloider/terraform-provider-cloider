# The subdomain roots are what cloider_options offers
data "cloider_options" "current" {}

resource "cloider_domain" "preview" {
  name = "acme-preview.${data.cloider_options.current.subdomain_roots[0]}"
}
