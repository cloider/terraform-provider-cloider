# A free subdomain
resource "cloider_domain" "app" {
  name = "acme.cloider.app"
}

# A domain you own, pointed at Cloider's nameservers at its registrar
resource "cloider_domain" "example" {
  name = "example.com"
}

data "cloider_options" "current" {}

output "nameservers" {
  value = data.cloider_options.current.nameservers
}
