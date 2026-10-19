resource "cloider_cluster" "web" {
  name = "web"
}

provider "kubernetes" {
  host                   = cloider_cluster.web.kube_config.host
  cluster_ca_certificate = cloider_cluster.web.kube_config.cluster_ca_certificate
  client_certificate     = cloider_cluster.web.kube_config.client_certificate
  client_key             = cloider_cluster.web.kube_config.client_key
}

resource "kubernetes_namespace_v1" "shop" {
  metadata {
    name = "shop"
  }
}
