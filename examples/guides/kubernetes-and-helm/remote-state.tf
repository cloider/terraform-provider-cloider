# In the configuration that deploys into the cluster, reading the outputs of
# the one that created it. Use the backend that configuration's state is in.
data "terraform_remote_state" "platform" {
  backend = "local"

  config = {
    path = "../platform/terraform.tfstate"
  }
}

locals {
  kube = data.terraform_remote_state.platform.outputs.kube_config
}

provider "kubernetes" {
  host                   = local.kube.host
  cluster_ca_certificate = local.kube.cluster_ca_certificate
  client_certificate     = local.kube.client_certificate
  client_key             = local.kube.client_key
}
