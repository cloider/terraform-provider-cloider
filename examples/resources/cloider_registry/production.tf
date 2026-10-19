# Tags can't be pushed over or deleted, so a deployed tag always means the same image
resource "cloider_registry" "releases" {
  name             = "acme-releases"
  storage_limit_gi = 20
  immutable_tags   = true
}
