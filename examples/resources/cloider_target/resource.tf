resource "cloider_domain" "shop" {
  name = "acme-shop.cloider.app"
}

# https://acme-shop.cloider.app/ reaches port 80 of the web Service in the
# shop namespace
resource "cloider_target" "shop" {
  domain     = cloider_domain.shop.name
  cluster_id = cloider_cluster.web.id
  namespace  = "shop"
  service    = "web"
  port       = 80
}
