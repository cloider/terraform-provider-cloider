data "cloider_options" "current" {}

locals {
  postgresql_versions = one([
    for e in data.cloider_options.current.database_engines : e.versions if e.engine == "postgresql"
  ])
}

resource "cloider_database" "orders" {
  name    = "orders"
  engine  = "postgresql"
  version = local.postgresql_versions[0]

  lifecycle {
    ignore_changes = [version]
  }
}
