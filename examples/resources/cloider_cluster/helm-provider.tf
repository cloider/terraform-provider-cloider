resource "cloider_cluster" "web" {
  name = "web"
}

# The helm provider 3.x takes kubernetes as an attribute
provider "helm" {
  kubernetes = {
    host                   = cloider_cluster.web.kube_config.host
    cluster_ca_certificate = cloider_cluster.web.kube_config.cluster_ca_certificate
    client_certificate     = cloider_cluster.web.kube_config.client_certificate
    client_key             = cloider_cluster.web.kube_config.client_key
  }
}

resource "helm_release" "podinfo" {
  name             = "podinfo"
  repository       = "https://stefanprodan.github.io/podinfo"
  chart            = "podinfo"
  namespace        = "podinfo"
  create_namespace = true
}
