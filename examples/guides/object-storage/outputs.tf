output "s3_access_key" {
  value = cloider_bucket_key.app.access_key
}

output "s3_secret_key" {
  value     = cloider_bucket_key.app.secret_key
  sensitive = true
}
