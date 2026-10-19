# Moving to a larger plan resizes the database in place, with a restart. A
# smaller plan can't be resized to, so it replaces the database and its data.
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "standard"
}
