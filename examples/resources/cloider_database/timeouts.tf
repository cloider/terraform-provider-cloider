resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "pro"

  timeouts = {
    create = "40m"
    update = "40m"
  }
}
