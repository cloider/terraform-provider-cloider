output "cluster_id" {
  value = cloider_cluster.this.id
}

output "kube_config" {
  value     = cloider_cluster.this.kube_config
  sensitive = true
}

output "database_url" {
  description = "For apps in the cluster."
  value       = "postgresql://${cloider_database.this.user}:${urlencode(cloider_database.this.password)}@${cloider_database.this.private_host}:${cloider_database.this.port}/${cloider_database.this.name}?sslmode=require"
  sensitive   = true
}

output "bucket" {
  value = {
    name     = cloider_bucket.uploads.name
    endpoint = cloider_bucket.uploads.endpoint
    region   = cloider_bucket.uploads.region
  }
}

output "bucket_access_key" {
  value = cloider_bucket_key.app.access_key
}

output "bucket_secret_key" {
  value     = cloider_bucket_key.app.secret_key
  sensitive = true
}
