# Lets an existing cluster reach a new database
data "cloider_cluster" "web" {
  name = "web"
}

resource "cloider_database" "orders" {
  name            = "orders"
  engine          = "postgresql"
  trusted_sources = ["cluster:${data.cloider_cluster.web.id}"]
}
