resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  user   = "orders_app"
  label  = "orders-production"
}
