variable "domain" {
  description = "A domain you own, such as example.com."
  type        = string
}

variable "web_ipv4" {
  description = "The addresses the domain and www point at."
  type        = list(string)
}

variable "mail_servers" {
  description = "The MX values, a priority and a host each, such as \"10 mx1.mail.example.net.\"."
  type        = list(string)
  default     = []
}

variable "spf" {
  description = "The domain's SPF policy, such as \"v=spf1 include:mail.example.net ~all\". Null for none."
  type        = string
  default     = null
}

variable "dmarc_report_email" {
  description = "Where DMARC reports go. Null for no DMARC record."
  type        = string
  default     = null
}

variable "extra_records" {
  description = "Any other records, by a key of your choice."
  type = map(object({
    name   = string
    type   = string
    values = list(string)
    ttl    = optional(number, 3600)
  }))
  default = {}
}
