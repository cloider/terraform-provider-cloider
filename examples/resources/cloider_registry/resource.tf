resource "cloider_registry" "images" {
  name             = "acme"
  storage_limit_gi = 10
  immutable_tags   = true
}
