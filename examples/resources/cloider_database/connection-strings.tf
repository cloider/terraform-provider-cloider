# The connection strings the dashboard shows, for clients outside Cloider.
# Use private_host instead of host from your clusters and apps.
locals {
  pg    = cloider_database.orders
  mysql = cloider_database.accounts
  mongo = cloider_database.events
  redis = cloider_database.cache
}

output "postgresql_url" {
  value     = "postgresql://${local.pg.user}:${urlencode(local.pg.password)}@${local.pg.host}:${local.pg.port}/${local.pg.name}?sslmode=require"
  sensitive = true
}

output "mysql_url" {
  value     = "mysql://${local.mysql.user}:${urlencode(local.mysql.password)}@${local.mysql.host}:${local.mysql.port}/${local.mysql.name}"
  sensitive = true
}

output "mongodb_url" {
  value     = "mongodb://${local.mongo.user}:${urlencode(local.mongo.password)}@${local.mongo.host}:${local.mongo.port}/${local.mongo.name}?directConnection=true&tls=true"
  sensitive = true
}

# Redis is TLS only unless ssl = false, then the scheme is redis://
output "redis_url" {
  value     = "rediss://${local.redis.user}:${urlencode(local.redis.password)}@${local.redis.host}:${local.redis.port}/0"
  sensitive = true
}
