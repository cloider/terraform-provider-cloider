# The key as environment variables for a workload in a Cloider cluster,
# with the kubernetes provider configured as in the cloider_cluster docs
resource "kubernetes_secret_v1" "s3" {
  metadata {
    name      = "s3"
    namespace = "default"
  }

  data = {
    AWS_ACCESS_KEY_ID     = cloider_bucket_key.deploy.access_key
    AWS_SECRET_ACCESS_KEY = cloider_bucket_key.deploy.secret_key
    AWS_REGION            = cloider_bucket.assets.region
    AWS_ENDPOINT_URL_S3   = cloider_bucket.assets.endpoint
    S3_BUCKET             = cloider_bucket.assets.name
  }
}
