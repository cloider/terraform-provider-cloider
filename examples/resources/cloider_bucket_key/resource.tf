resource "cloider_bucket" "assets" {
  name = "acme-assets"
}

# Reads and writes one bucket
resource "cloider_bucket_key" "deploy" {
  label   = "CI deploy"
  buckets = [cloider_bucket.assets.id]
}
