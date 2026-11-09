resource "cloider_domain_verification" "example" {
  domain = cloider_domain.example.name

  timeouts = {
    create = "30m"
  }
}
