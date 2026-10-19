# A domain you own. Point it at these nameservers at its registrar.
resource "cloider_domain" "example" {
  name = "example.com"
}

data "cloider_options" "current" {}

output "nameservers" {
  value = data.cloider_options.current.nameservers
}
