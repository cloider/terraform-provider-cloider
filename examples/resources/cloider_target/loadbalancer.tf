resource "cloider_domain" "games" {
  name = "play.example.com"
}

# play.example.com resolves to the IP of the server LoadBalancer in the games
# namespace, on whatever ports it listens on
resource "cloider_target" "games" {
  domain       = cloider_domain.games.name
  cluster_id   = cloider_cluster.web.id
  loadbalancer = "games/server"
}
