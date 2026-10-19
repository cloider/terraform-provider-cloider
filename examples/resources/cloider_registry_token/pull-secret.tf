resource "cloider_registry_token" "pull" {
  label     = "web cluster pull"
  projects  = [cloider_registry.images.id]
  read_only = true
}

# An image pull secret for the kubernetes provider, see the cloider_cluster docs
resource "kubernetes_secret_v1" "registry" {
  metadata {
    name      = "cloider-registry"
    namespace = "default"
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        (cloider_registry.images.host) = {
          username = cloider_registry_token.pull.username
          password = cloider_registry_token.pull.secret
          auth     = base64encode("${cloider_registry_token.pull.username}:${cloider_registry_token.pull.secret}")
        }
      }
    })
  }
}
