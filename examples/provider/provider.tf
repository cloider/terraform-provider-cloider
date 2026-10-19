terraform {
  required_providers {
    cloider = {
      source  = "cloider/cloider"
      version = "~> 0.1"
    }
  }
}

# Reads the token from CLOIDER_TOKEN, or the one `cloider login` saved
provider "cloider" {}

# A Kubernetes cluster
resource "cloider_cluster" "web" {
  name = "web"
}

# A PostgreSQL database only that cluster can reach
resource "cloider_database" "orders" {
  name            = "orders"
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}
