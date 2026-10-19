---
page_title: "Deploying to a Cluster with the Kubernetes and Helm Providers"
subcategory: ""
description: |-
  Deploy apps and Helm charts into a Cloider cluster from Terraform.
---

# Deploying to a Cluster with the Kubernetes and Helm Providers

A [`cloider_cluster`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/cluster)'s
`kube_config` holds what the [`kubernetes`](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs)
and [`helm`](https://registry.terraform.io/providers/hashicorp/helm/latest/docs) providers need to
deploy into it. This guide runs an app from a Cloider registry project against a Cloider database, and
serves it on a domain.
The whole configuration is in
[`examples/kubernetes-app`](https://github.com/cloider/terraform-provider-cloider/tree/main/examples/kubernetes-app).

## Configure the Providers

The cluster's kubeconfig authenticates with a client certificate, as the cluster's admin:

```terraform
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
```

-> **Note:** The helm provider 3.x takes `kubernetes` as an attribute, `kubernetes = { ... }`. The 2.x
releases took it as a block.

## One Configuration or Two

A provider configured from a resource can't reach the cluster before the cluster exists, so choose one
of these.

### One Configuration, Applied in Two Steps

Create the cluster on its own first, then everything else. Later applies need no `-target`. This suits
a cluster that is rarely replaced, such as one per app:

```shell
terraform apply -target=cloider_cluster.app
terraform apply
```

### Two Configurations

One configuration creates the Cloider resources, a second one deploys into the cluster. HashiCorp
recommends this for the kubernetes provider, since a plan that replaces the cluster can't plan what
runs in it. The second one looks the cluster up with the
[`cloider_cluster`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/data-sources/cluster)
data source:

```terraform
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
```

Or the first one outputs the kubeconfig:

```terraform
# In the configuration that creates the cluster
output "kube_config" {
  value     = cloider_cluster.app.kube_config
  sensitive = true
}
```

and the second one reads it with `terraform_remote_state`:

```terraform
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
```

## Deploy an App

The app's image is pulled from a [`cloider_registry`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/registry)
project with a read-only token, handed to the cluster as an image pull secret. The database only
trusts the cluster, which reaches it at `private_host`:

```terraform
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
```

## Install a Helm Chart

```terraform
resource "helm_release" "podinfo" {
  name             = "podinfo"
  repository       = "https://stefanprodan.github.io/podinfo"
  chart            = "podinfo"
  version          = "6.15.0"
  namespace        = "podinfo"
  create_namespace = true

  set = [
    {
      name  = "replicaCount"
      value = "2"
    },
  ]
}
```

## Reach the App from the Internet

Ingress objects in a Cloider cluster aren't served. A
[`cloider_target`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/target)
routes a domain to a Service instead, over HTTPS, as the end of the configuration above does with a
free `cloider.app` subdomain. A target can also route single paths, such as `/api`, to Services of
their own.
