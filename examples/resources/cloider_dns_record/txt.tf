# Each value is a TXT record of its own, so an SPF policy and verification
# tokens at the same name go in one resource
resource "cloider_dns_record" "apex_txt" {
  domain = "example.com"
  type   = "TXT"
  values = [
    "v=spf1 include:mail.example.net ~all",
    "google-site-verification=rXOxyZounnZasA8Z7oaD3c14JdjS9aKSWvsR1EbUSIQ",
  ]
  ttl = 3600
}

# A value over 255 characters, such as a DKIM key, is fine as one value
resource "cloider_dns_record" "dkim" {
  domain = "example.com"
  type   = "TXT"
  name   = "mail._domainkey"
  values = ["v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu5..."]
  ttl    = 3600
}
