provider "cloider" {}

data "cloider_options" "current" {}

resource "cloider_domain" "this" {
  name = var.domain
}

resource "cloider_dns_record" "apex" {
  domain = cloider_domain.this.name
  type   = "A"
  values = var.web_ipv4
  ttl    = 300
}

resource "cloider_dns_record" "www" {
  domain = cloider_domain.this.name
  type   = "CNAME"
  name   = "www"
  values = ["${cloider_domain.this.name}."]
  ttl    = 3600
}

resource "cloider_dns_record" "mx" {
  count = length(var.mail_servers) > 0 ? 1 : 0

  domain = cloider_domain.this.name
  type   = "MX"
  values = var.mail_servers
  ttl    = 3600
}

resource "cloider_dns_record" "spf" {
  count = var.spf != null ? 1 : 0

  domain = cloider_domain.this.name
  type   = "TXT"
  values = [var.spf]
  ttl    = 3600
}

resource "cloider_dns_record" "dmarc" {
  count = var.dmarc_report_email != null ? 1 : 0

  domain = cloider_domain.this.name
  type   = "TXT"
  name   = "_dmarc"
  values = ["v=DMARC1; p=quarantine; rua=mailto:${var.dmarc_report_email}"]
  ttl    = 3600
}

resource "cloider_dns_record" "extra" {
  for_each = var.extra_records

  domain = cloider_domain.this.name
  name   = each.value.name
  type   = each.value.type
  values = each.value.values
  ttl    = each.value.ttl
}
