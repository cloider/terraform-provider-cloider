variable "staging_paused" {
  description = "Pauses the staging cluster, which keeps its state and isn't billed for compute."
  type        = bool
  default     = false
}

resource "cloider_cluster" "staging" {
  name   = "staging"
  paused = var.staging_paused
}
