# In the configuration that deploys into the cluster
data "cloider_cluster" "app" {
  name = "shop"
}

provider "kubernetes" {
  host                   = data.cloider_cluster.app.kube_config.host
  cluster_ca_certificate = data.cloider_cluster.app.kube_config.cluster_ca_certificate
  client_certificate     = data.cloider_cluster.app.kube_config.client_certificate
  client_key             = data.cloider_cluster.app.kube_config.client_key
}
