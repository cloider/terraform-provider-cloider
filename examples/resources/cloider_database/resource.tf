resource "cloider_cluster" "web" {
  name = "web"
}

resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "starter"

  # The cluster, and the office network
  trusted_sources = [
    "cluster:${cloider_cluster.web.id}",
    "203.0.113.0/24",
  ]
}

output "orders_url" {
  value     = "postgresql://${cloider_database.orders.user}:${cloider_database.orders.password}@${cloider_database.orders.private_host}:${cloider_database.orders.port}/${cloider_database.orders.name}"
  sensitive = true
}
