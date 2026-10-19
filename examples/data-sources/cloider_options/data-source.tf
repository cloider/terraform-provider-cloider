data "cloider_options" "current" {}

output "kubernetes_versions" {
  value = data.cloider_options.current.cluster_versions
}
