# Reaches every project, those created later included
resource "cloider_registry_token" "mirror" {
  label        = "Backup mirror"
  all_projects = true
  read_only    = true
}
