resource "helm_release" "podinfo" {
  name             = "podinfo"
  repository       = "https://stefanprodan.github.io/podinfo"
  chart            = "podinfo"
  version          = "6.15.0"
  namespace        = "podinfo"
  create_namespace = true

  set = [
    {
      name  = "replicaCount"
      value = "2"
    },
  ]
}
