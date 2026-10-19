provider "cloider" {}

# Cloider resources

resource "cloider_cluster" "app" {
  name = var.name
}

resource "cloider_database" "app" {
  name            = var.name
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.app.id}"]
}

data "cloider_options" "current" {}

resource "cloider_domain" "app" {
  name = "${var.subdomain}.${data.cloider_options.current.subdomain_roots[0]}"
}

resource "cloider_registry" "app" {
  name = var.registry_project
}

# For CI to push images with
resource "cloider_registry_token" "push" {
  label           = "${var.name} CI push"
  projects        = [cloider_registry.app.id]
  expires_in_days = 365
}

# For the cluster to pull them with
resource "cloider_registry_token" "pull" {
  label     = "${var.name} cluster pull"
  projects  = [cloider_registry.app.id]
  read_only = true
}

# Kubernetes resources, in the cluster above

provider "kubernetes" {
  host                   = cloider_cluster.app.kube_config.host
  cluster_ca_certificate = cloider_cluster.app.kube_config.cluster_ca_certificate
  client_certificate     = cloider_cluster.app.kube_config.client_certificate
  client_key             = cloider_cluster.app.kube_config.client_key
}

resource "kubernetes_namespace_v1" "app" {
  metadata {
    name = var.name
  }
}

resource "kubernetes_secret_v1" "registry" {
  metadata {
    name      = "cloider-registry"
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        (cloider_registry.app.host) = {
          username = cloider_registry_token.pull.username
          password = cloider_registry_token.pull.secret
          auth     = base64encode("${cloider_registry_token.pull.username}:${cloider_registry_token.pull.secret}")
        }
      }
    })
  }
}

resource "kubernetes_secret_v1" "database" {
  metadata {
    name      = "database"
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }

  data = {
    DATABASE_URL = "postgresql://${cloider_database.app.user}:${urlencode(cloider_database.app.password)}@${cloider_database.app.private_host}:${cloider_database.app.port}/${cloider_database.app.name}?sslmode=require"
  }
}

resource "kubernetes_deployment_v1" "app" {
  metadata {
    name      = var.name
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }

  # Applying doesn't wait for the pods, which can't start before the image is pushed
  wait_for_rollout = false

  spec {
    replicas = var.replicas

    selector {
      match_labels = {
        app = var.name
      }
    }

    template {
      metadata {
        labels = {
          app = var.name
        }
      }

      spec {
        image_pull_secrets {
          name = kubernetes_secret_v1.registry.metadata[0].name
        }

        container {
          name  = var.name
          image = "${cloider_registry.app.url}/${var.name}:${var.image_tag}"

          port {
            container_port = 8080
          }

          env_from {
            secret_ref {
              name = kubernetes_secret_v1.database.metadata[0].name
            }
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "app" {
  metadata {
    name      = var.name
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }

  spec {
    selector = {
      app = var.name
    }

    port {
      port        = 80
      target_port = 8080
    }
  }
}

# Serves the Service at https://<subdomain>.cloider.app

resource "cloider_target" "app" {
  domain     = cloider_domain.app.name
  cluster_id = cloider_cluster.app.id
  namespace  = kubernetes_service_v1.app.metadata[0].namespace
  service    = kubernetes_service_v1.app.metadata[0].name
  port       = 80
}
