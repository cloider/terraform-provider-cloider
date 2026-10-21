# Terraform provider for Cloider

Manages [Cloider](https://cloider.com) clusters, databases, buckets and their keys, container
registry projects and their tokens, domains, DNS records and the targets that serve a domain from a
cluster, with Terraform or OpenTofu.

```hcl
terraform {
  required_providers {
    cloider = {
      source = "cloider/cloider"
    }
  }
}

provider "cloider" {}

resource "cloider_cluster" "web" {
  name = "web"
}

resource "cloider_database" "orders" {
  name            = "orders"
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}
```

The provider takes its API token from `CLOIDER_TOKEN`, or from `cloider login` on your own machine.

## Documentation

The full reference is on the [Terraform Registry](https://registry.terraform.io/providers/cloider/cloider/latest/docs),
with examples for every resource and these guides:

* [Getting Started](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/getting-started)
* [Deploying to a Cluster with the Kubernetes and Helm Providers](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/kubernetes-and-helm)
* [Using Buckets with S3 Tools](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/object-storage)
* [Storing Terraform State in a Bucket](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/state-backend)
* [Importing Existing Resources](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/importing-resources)

## Examples

Complete configurations to start from are in [`examples/`](examples):

| Example                                                  | What it creates                                                              |
|----------------------------------------------------------|------------------------------------------------------------------------------|
| [getting-started](examples/getting-started)              | A cluster, a database only it can reach, and a bucket with a key.            |
| [kubernetes-app](examples/kubernetes-app)                | An app on its own cluster, with its registry project, database and domain.   |
| [custom-domain](examples/custom-domain)                  | DNS for a domain you own: web, mail and any other records.                   |
| [multiple-environments](examples/multiple-environments)  | Staging and production from one module.                                      |

## Requirements

* [Terraform](https://developer.hashicorp.com/terraform/install) 1.0 or later, or
  [OpenTofu](https://opentofu.org/docs/intro/install/) 1.6 or later. `import` blocks need Terraform 1.5.
* A [Cloider](https://cloider.com) account and an API token, from Access Control, API Tokens in the
  dashboard.

Each release is published to this repository from Cloider's own, so pull requests can't be merged
here. Issues and feature requests are welcome.
