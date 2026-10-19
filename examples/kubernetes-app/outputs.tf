output "registry_url" {
  description = "Push the app's image as <registry_url>/<name>:<tag>."
  value       = cloider_registry.app.url
}

output "registry_push_username" {
  value = cloider_registry_token.push.username
}

output "registry_push_password" {
  value     = cloider_registry_token.push.secret
  sensitive = true
}

output "kubeconfig" {
  value     = cloider_cluster.app.kube_config.raw
  sensitive = true
}

output "url" {
  value = "https://${cloider_domain.app.name}"
}
