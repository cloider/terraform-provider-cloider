# Anyone may pull without a token, pushing still needs one
resource "cloider_registry" "oss" {
  name   = "acme-oss"
  public = true
}
