# Hands the token to a GitHub Actions workflow with the integrations/github provider
resource "github_actions_secret" "registry_username" {
  repository      = "web"
  secret_name     = "CLOIDER_REGISTRY_USERNAME"
  plaintext_value = cloider_registry_token.ci.username
}

resource "github_actions_secret" "registry_password" {
  repository      = "web"
  secret_name     = "CLOIDER_REGISTRY_PASSWORD"
  plaintext_value = cloider_registry_token.ci.secret
}
