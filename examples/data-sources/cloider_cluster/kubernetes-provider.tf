# In a configuration of its own, deploying into a cluster another one created
data "cloider_cluster" "web" {
  id = var.cluster_id
}

provider "kubernetes" {
  host                   = data.cloider_cluster.web.kube_config.host
  cluster_ca_certificate = data.cloider_cluster.web.kube_config.cluster_ca_certificate
  client_certificate     = data.cloider_cluster.web.kube_config.client_certificate
  client_key             = data.cloider_cluster.web.kube_config.client_key
}

provider "helm" {
  kubernetes = {
    host                   = data.cloider_cluster.web.kube_config.host
    cluster_ca_certificate = data.cloider_cluster.web.kube_config.cluster_ca_certificate
    client_certificate     = data.cloider_cluster.web.kube_config.client_certificate
    client_key             = data.cloider_cluster.web.kube_config.client_key
  }
}
