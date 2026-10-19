data "cloider_cluster" "web" {
  name = "web"
}

output "web_cluster_status" {
  value = data.cloider_cluster.web.status
}
