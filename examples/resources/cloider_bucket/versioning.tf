# Keeps the previous versions of overwritten and deleted files
resource "cloider_bucket" "backups" {
  name       = "acme-backups"
  versioning = true
}
