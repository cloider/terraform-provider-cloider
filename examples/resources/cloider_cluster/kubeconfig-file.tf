resource "cloider_cluster" "web" {
  name = "web"
}

# For kubectl: KUBECONFIG=./kubeconfig-web.yaml kubectl get pods -A
resource "local_sensitive_file" "kubeconfig" {
  filename        = "${path.module}/kubeconfig-web.yaml"
  content         = cloider_cluster.web.kube_config.raw
  file_permission = "0600"
}
