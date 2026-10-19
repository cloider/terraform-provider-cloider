resource "cloider_dns_record" "mx" {
  domain = "example.com"
  type   = "MX"
  values = ["10 mx1.mail.example.net.", "20 mx2.mail.example.net."]
  ttl    = 3600
}

resource "cloider_dns_record" "spf" {
  domain = "example.com"
  type   = "TXT"
  values = ["v=spf1 include:mail.example.net ~all"]
  ttl    = 3600
}

resource "cloider_dns_record" "dmarc" {
  domain = "example.com"
  type   = "TXT"
  name   = "_dmarc"
  values = ["v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"]
  ttl    = 3600
}
