data "cloider_options" "current" {}

resource "cloider_cluster" "web" {
  name    = "web"
  version = data.cloider_options.current.cluster_versions[0]
}
