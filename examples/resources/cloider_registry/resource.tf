resource "cloider_registry" "images" {
  name = "acme"
}

# Push as <url>/<repository>:<tag>, such as acme/web:1.4.0
output "image_prefix" {
  value = cloider_registry.images.url
}
