# Replaces the token every 90 days, a month before it would expire. The new
# token is created before the old one is deleted.
resource "time_rotating" "ci" {
  rotation_days = 90
}

resource "cloider_registry_token" "ci" {
  label           = "CI push"
  projects        = [cloider_registry.images.id]
  expires_in_days = 120

  lifecycle {
    create_before_destroy = true
    replace_triggered_by  = [time_rotating.ci]
  }
}
