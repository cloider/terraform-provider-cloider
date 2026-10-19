provider "kubernetes" {
  host                   = cloider_cluster.app.kube_config.host
  cluster_ca_certificate = cloider_cluster.app.kube_config.cluster_ca_certificate
  client_certificate     = cloider_cluster.app.kube_config.client_certificate
  client_key             = cloider_cluster.app.kube_config.client_key
}

provider "helm" {
  kubernetes = {
    host                   = cloider_cluster.app.kube_config.host
    cluster_ca_certificate = cloider_cluster.app.kube_config.cluster_ca_certificate
    client_certificate     = cloider_cluster.app.kube_config.client_certificate
    client_key             = cloider_cluster.app.kube_config.client_key
  }
}
