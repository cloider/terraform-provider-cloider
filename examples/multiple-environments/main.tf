provider "cloider" {}

module "staging" {
  source = "./modules/environment"

  name             = "staging"
  bucket_prefix    = var.bucket_prefix
  database_plan    = "hobby"
  trusted_networks = var.office_network != null ? [var.office_network] : []
}

module "production" {
  source = "./modules/environment"

  name          = "production"
  bucket_prefix = var.bucket_prefix
  database_plan = "standard"
}
