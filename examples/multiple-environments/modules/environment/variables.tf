variable "name" {
  description = "The environment's name, such as staging. Used in every resource's name."
  type        = string
}

variable "bucket_prefix" {
  description = "Prefix of the bucket's name, which has to be unique across every Cloider account."
  type        = string
}

variable "database_plan" {
  description = "The database's plan."
  type        = string
  default     = "hobby"
}

variable "paused" {
  description = "Pauses the cluster and the database, keeping their state."
  type        = bool
  default     = false
}

variable "trusted_networks" {
  description = "IPv4 CIDRs that may reach the database besides the cluster, such as an office network."
  type        = list(string)
  default     = []
}
