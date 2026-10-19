data "cloider_options" "current" {}

# A cluster isn't upgraded in place: a new version replaces it. Ignoring
# changes keeps the version it was created with.
resource "cloider_cluster" "web" {
  name    = "web"
  version = data.cloider_options.current.cluster_versions[0]

  lifecycle {
    ignore_changes = [version]
  }
}
