# Routes to a Service the kubernetes provider manages, once it exists
resource "kubernetes_service_v1" "web" {
  metadata {
    name      = "web"
    namespace = "shop"
  }

  spec {
    selector = {
      app = "web"
    }

    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "cloider_target" "shop" {
  domain     = cloider_domain.shop.name
  cluster_id = cloider_cluster.web.id
  namespace  = kubernetes_service_v1.web.metadata[0].namespace
  service    = kubernetes_service_v1.web.metadata[0].name
  port       = 80
}
