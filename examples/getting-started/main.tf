# Reads the token from CLOIDER_TOKEN, or the one `cloider login` saved
provider "cloider" {}

resource "cloider_cluster" "main" {
  name = "main"
}

# Only reachable from the cluster
resource "cloider_database" "app" {
  name            = "app"
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.main.id}"]
}

resource "cloider_bucket" "uploads" {
  name = "${var.prefix}-uploads"
}

resource "cloider_bucket_key" "app" {
  label   = "app"
  buckets = [cloider_bucket.uploads.id]
}
