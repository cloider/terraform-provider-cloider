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
