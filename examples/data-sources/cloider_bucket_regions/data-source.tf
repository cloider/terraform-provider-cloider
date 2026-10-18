data "cloider_bucket_regions" "all" {}

resource "cloider_bucket" "assets" {
  name   = "acme-assets"
  region = data.cloider_bucket_regions.all.regions[0].code
}
