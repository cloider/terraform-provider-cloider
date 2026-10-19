output "kubeconfig" {
  description = "terraform output -raw kubeconfig > kubeconfig.yaml"
  value       = cloider_cluster.main.kube_config.raw
  sensitive   = true
}

output "database_host" {
  description = "The host to reach the database at from the cluster."
  value       = cloider_database.app.private_host
}

output "database_port" {
  value = cloider_database.app.port
}

output "database_user" {
  value = cloider_database.app.user
}

output "database_password" {
  value     = cloider_database.app.password
  sensitive = true
}

output "s3_endpoint" {
  value = cloider_bucket.uploads.endpoint
}

output "s3_access_key" {
  value = cloider_bucket_key.app.access_key
}

output "s3_secret_key" {
  value     = cloider_bucket_key.app.secret_key
  sensitive = true
}
