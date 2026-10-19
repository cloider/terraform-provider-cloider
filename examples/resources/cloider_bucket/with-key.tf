resource "cloider_bucket" "uploads" {
  name = "acme-uploads"
}

resource "cloider_bucket_key" "app" {
  label   = "app"
  buckets = [cloider_bucket.uploads.id]
}

# The settings any S3 client takes
output "s3_environment" {
  value = {
    AWS_ACCESS_KEY_ID   = cloider_bucket_key.app.access_key
    AWS_REGION          = cloider_bucket.uploads.region
    AWS_ENDPOINT_URL_S3 = cloider_bucket.uploads.endpoint
    S3_BUCKET           = cloider_bucket.uploads.name
  }
}
