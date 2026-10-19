# Redis is TLS only by default. ssl = false serves plain redis:// instead,
# for clients without TLS support. Keep its trusted sources tight.
resource "cloider_database" "cache" {
  name            = "cache"
  engine          = "redis"
  ssl             = false
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}
