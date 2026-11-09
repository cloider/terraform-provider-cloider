output "verification_record" {
  description = "Add this TXT record where the domain's DNS is hosted now, to verify it."
  value = {
    type  = "TXT"
    name  = cloider_domain.this.verification_record_name
    value = cloider_domain.this.verification_record_value
  }
}

output "nameservers" {
  description = "Set these as the domain's nameservers at its registrar, once it's verified."
  value       = data.cloider_options.current.nameservers
}

output "status" {
  value = cloider_domain.this.status
}
