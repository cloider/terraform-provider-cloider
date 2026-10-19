resource "cloider_domain" "example" {
  name = "example.com"
}

resource "cloider_dns_record" "www" {
  domain = cloider_domain.example.name
  type   = "CNAME"
  name   = "www"
  values = ["example.com."]
  ttl    = 3600
}
