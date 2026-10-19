# Which certificate authorities may issue certificates for the domain. Cloider's
# certificates for a cloider_target come from Let's Encrypt, so keep it allowed.
resource "cloider_dns_record" "caa" {
  domain = "example.com"
  type   = "CAA"
  values = [
    "0 issue \"letsencrypt.org\"",
    "0 iodef \"mailto:security@example.com\"",
  ]
  ttl = 3600
}
