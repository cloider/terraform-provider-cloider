output "nameservers" {
  description = "Set these as the domain's nameservers at its registrar."
  value       = data.cloider_options.current.nameservers
}

output "status" {
  value = cloider_domain.this.status
}
