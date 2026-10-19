output "s3_endpoint" {
  value = cloider_bucket.assets.endpoint
}

output "s3_access_key" {
  value = cloider_bucket_key.deploy.access_key
}

# terraform output -raw s3_secret_key
output "s3_secret_key" {
  value     = cloider_bucket_key.deploy.secret_key
  sensitive = true
}
