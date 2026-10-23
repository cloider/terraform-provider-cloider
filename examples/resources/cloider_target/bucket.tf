# A bucket's static website, on the whole domain: path stays "/"
resource "cloider_target" "site" {
  domain    = cloider_domain.shop.name
  bucket_id = cloider_bucket.site.id
}
