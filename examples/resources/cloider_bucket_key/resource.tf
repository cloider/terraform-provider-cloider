resource "cloider_bucket" "assets" {
  name = "acme-assets"
}

resource "cloider_bucket_key" "deploy" {
  label   = "CI deploy"
  buckets = [cloider_bucket.assets.id]
}

resource "cloider_bucket_key" "backups" {
  label       = "Backup reader"
  all_buckets = true
  read_only   = true
}
