# priority weight port target
resource "cloider_dns_record" "sip" {
  domain = "example.com"
  type   = "SRV"
  name   = "_sip._tcp"
  values = ["10 60 5060 sip1.example.com.", "20 40 5060 sip2.example.com."]
  ttl    = 3600
}
