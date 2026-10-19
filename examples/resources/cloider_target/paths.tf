# /api goes to the api Service, everything else to the web one
resource "cloider_target" "web" {
  domain     = cloider_domain.shop.name
  cluster_id = cloider_cluster.web.id
  namespace  = "shop"
  service    = "web"
  port       = 80
}

resource "cloider_target" "api" {
  domain     = cloider_domain.shop.name
  path       = "/api"
  cluster_id = cloider_cluster.web.id
  namespace  = "shop"
  service    = "api"
  port       = 8080
}
