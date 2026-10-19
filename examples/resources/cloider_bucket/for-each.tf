variable "environments" {
  type    = set(string)
  default = ["staging", "production"]
}

resource "cloider_bucket" "uploads" {
  for_each = var.environments

  name       = "acme-uploads-${each.key}"
  versioning = each.key == "production"
}
