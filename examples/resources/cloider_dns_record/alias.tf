# An ALIAS points the domain itself at a host name, where a CNAME can't go.
# Cloider answers A and AAAA lookups with the target's current addresses.
resource "cloider_dns_record" "apex_alias" {
  domain = "example.com"
  type   = "ALIAS"
  values = ["myapp.herokuapp.com"]
  ttl    = 300
}
