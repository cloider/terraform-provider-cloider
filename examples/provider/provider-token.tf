variable "cloider_token" {
  type      = string
  sensitive = true
}

provider "cloider" {
  token = var.cloider_token
}
