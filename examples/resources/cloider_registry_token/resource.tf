resource "cloider_registry" "images" {
  name = "acme"
}

resource "cloider_registry_token" "ci" {
  label           = "CI push"
  projects        = [cloider_registry.images.id]
  expires_in_days = 365
}

# docker login <host> -u <username> -p <secret>
output "registry_login" {
  value = {
    host     = cloider_registry.images.host
    username = cloider_registry_token.ci.username
  }
}

output "registry_password" {
  value     = cloider_registry_token.ci.secret
  sensitive = true
}
