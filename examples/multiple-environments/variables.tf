variable "bucket_prefix" {
  description = "Prefix of the buckets' names, which have to be unique across every Cloider account, such as your company's name."
  type        = string
}

variable "office_network" {
  description = "An IPv4 CIDR that may reach the staging database, such as 203.0.113.0/24. Null for none."
  type        = string
  default     = null
}
