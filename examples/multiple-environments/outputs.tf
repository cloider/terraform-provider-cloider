output "staging_database_url" {
  value     = module.staging.database_url
  sensitive = true
}

output "production_database_url" {
  value     = module.production.database_url
  sensitive = true
}

output "buckets" {
  value = {
    staging    = module.staging.bucket
    production = module.production.bucket
  }
}
