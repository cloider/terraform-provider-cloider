resource "cloider_cluster" "web" {
  name = "web"
}

resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"

  # The cluster, the office network and one CI runner
  trusted_sources = [
    "cluster:${cloider_cluster.web.id}",
    "203.0.113.0/24",
    "198.51.100.7/32",
  ]
}
