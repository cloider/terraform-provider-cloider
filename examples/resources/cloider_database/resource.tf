resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "starter"
}
