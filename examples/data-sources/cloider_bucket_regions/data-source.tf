data "cloider_bucket_regions" "all" {}

output "bucket_regions" {
  value = { for r in data.cloider_bucket_regions.all.regions : r.code => r.name }
}
