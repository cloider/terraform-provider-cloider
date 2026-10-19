# Private: every request needs a cloider_bucket_key
resource "cloider_bucket" "uploads" {
  name = "acme-uploads"
}
