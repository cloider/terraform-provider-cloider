locals {
  records = {
    "www/A"        = { name = "www", type = "A", values = ["203.0.113.10"] }
    "api/A"        = { name = "api", type = "A", values = ["203.0.113.20", "203.0.113.21"] }
    "status/CNAME" = { name = "status", type = "CNAME", values = ["status.example.net."] }
  }
}

resource "cloider_dns_record" "this" {
  for_each = local.records

  domain = "example.com"
  name   = each.value.name
  type   = each.value.type
  values = each.value.values
  ttl    = 3600
}
