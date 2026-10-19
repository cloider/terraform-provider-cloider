resource "cloider_cluster" "this" {
  name   = var.name
  paused = var.paused
}

resource "cloider_database" "this" {
  name   = var.name
  engine = "postgresql"
  plan   = var.database_plan
  paused = var.paused

  trusted_sources = concat(["cluster:${cloider_cluster.this.id}"], var.trusted_networks)
}

resource "cloider_bucket" "uploads" {
  name       = "${var.bucket_prefix}-${var.name}-uploads"
  versioning = true
}

resource "cloider_bucket_key" "app" {
  label   = "${var.name} app"
  buckets = [cloider_bucket.uploads.id]
}
