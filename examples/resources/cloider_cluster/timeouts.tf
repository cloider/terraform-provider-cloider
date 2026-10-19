resource "cloider_cluster" "web" {
  name = "web"

  timeouts = {
    create = "30m"
    delete = "30m"
  }
}
