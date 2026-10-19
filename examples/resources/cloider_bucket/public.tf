# Anyone may download a file with its link, without a key. Listing the
# bucket and changing files still need one.
resource "cloider_bucket" "assets" {
  name   = "acme-assets"
  public = true
}

# A file is read at <url>/<key>, such as .../logo.png
output "assets_url" {
  value = cloider_bucket.assets.url
}
