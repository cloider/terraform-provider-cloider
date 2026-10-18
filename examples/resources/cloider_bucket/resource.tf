resource "cloider_bucket" "assets" {
  name       = "acme-assets"
  public     = true
  versioning = true
}
