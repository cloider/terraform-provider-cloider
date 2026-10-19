data "cloider_bucket_regions" "all" {}

locals {
  regions = { for r in data.cloider_bucket_regions.all.regions : r.code => r }
}

resource "cloider_bucket" "assets" {
  name   = "acme-assets"
  region = local.regions["eu-central-2"].code
}

output "assets_endpoint" {
  value = local.regions["eu-central-2"].endpoint
}
