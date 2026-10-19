# A configuration of its own, whose state stays local
resource "cloider_bucket" "state" {
  name       = "acme-terraform-state"
  versioning = true

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloider_bucket_key" "state" {
  label   = "Terraform state"
  buckets = [cloider_bucket.state.id]
}

output "state_access_key" {
  value = cloider_bucket_key.state.access_key
}

output "state_secret_key" {
  value     = cloider_bucket_key.state.secret_key
  sensitive = true
}
