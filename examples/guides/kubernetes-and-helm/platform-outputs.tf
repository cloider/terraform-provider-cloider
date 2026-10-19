# In the configuration that creates the cluster
output "kube_config" {
  value     = cloider_cluster.app.kube_config
  sensitive = true
}
