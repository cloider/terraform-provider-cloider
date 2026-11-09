# A domain you own. Add this TXT record where its DNS is hosted now and verify
# it with cloider_domain_verification, then point it at these nameservers at its
# registrar.
resource "cloider_domain" "example" {
  name = "example.com"
}

data "cloider_options" "current" {}

output "verification_record" {
  value = {
    type  = "TXT"
    name  = cloider_domain.example.verification_record_name
    value = cloider_domain.example.verification_record_value
  }
}

output "nameservers" {
  value = data.cloider_options.current.nameservers
}
