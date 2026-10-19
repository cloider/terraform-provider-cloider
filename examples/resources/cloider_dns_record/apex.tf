# name left out is the domain itself, example.com
resource "cloider_dns_record" "apex" {
  domain = "example.com"
  type   = "A"
  values = ["203.0.113.10"]
  ttl    = 300
}

resource "cloider_dns_record" "apex_ipv6" {
  domain = "example.com"
  type   = "AAAA"
  values = ["2001:db8::10"]
  ttl    = 300
}
