# https://acme-shop.cloider.app/docs/intro reaches the docs Service as /intro
resource "cloider_target" "docs" {
  domain      = cloider_domain.shop.name
  path        = "/docs"
  target_path = "/"
  cluster_id  = cloider_cluster.web.id
  namespace   = "docs"
  service     = "docs"
  port        = 80
}
