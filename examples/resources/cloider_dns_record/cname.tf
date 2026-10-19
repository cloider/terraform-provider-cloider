# A CNAME takes one value, and no other record can share its name
resource "cloider_dns_record" "docs" {
  domain = "example.com"
  type   = "CNAME"
  name   = "docs"
  values = ["example.github.io."]
  ttl    = 3600
}
