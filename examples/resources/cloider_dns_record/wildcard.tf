# Answers for any name under example.com that has no record of its own
resource "cloider_dns_record" "wildcard" {
  domain = "example.com"
  type   = "A"
  name   = "*"
  values = ["203.0.113.10"]
  ttl    = 300
}
