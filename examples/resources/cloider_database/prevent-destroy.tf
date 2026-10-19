# Refuses any plan that would delete the database or replace it, such as a
# smaller plan or another version
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "standard"

  lifecycle {
    prevent_destroy = true
  }
}
