resource "cloider_database" "orders" {
  name            = "orders"
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}

# The credentials as a Secret in the cluster, with the kubernetes provider
# configured as in the cloider_cluster docs
resource "kubernetes_secret_v1" "orders_db" {
  metadata {
    name      = "orders-db"
    namespace = "default"
  }

  data = {
    DATABASE_URL = "postgresql://${cloider_database.orders.user}:${urlencode(cloider_database.orders.password)}@${cloider_database.orders.private_host}:${cloider_database.orders.port}/${cloider_database.orders.name}?sslmode=require"

    # For clients that verify the server's certificate
    "ca.crt" = cloider_database.orders.ca_certificate
  }
}
