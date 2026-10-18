resource "cloider_domain" "example" {
  name = "example.com"
}

resource "cloider_dns_record" "www" {
  domain = cloider_domain.example.name
  type   = "A"
  name   = "www"
  values = ["203.0.113.10", "203.0.113.11"]
  ttl    = 3600
}

resource "cloider_dns_record" "mail" {
  domain = cloider_domain.example.name
  type   = "MX"
  values = ["10 mx1.example.net.", "20 mx2.example.net."]
  ttl    = 3600
}
