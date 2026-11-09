resource "cloider_domain" "example" {
  name = "example.com"
}

# The TXT record, at the DNS provider that hosts example.com now
data "aws_route53_zone" "example" {
  name = "example.com"
}

resource "aws_route53_record" "cloider_verify" {
  zone_id = data.aws_route53_zone.example.zone_id
  name    = cloider_domain.example.verification_record_name
  type    = "TXT"
  ttl     = 300
  records = [cloider_domain.example.verification_record_value]
}

# Waits until Cloider finds the record
resource "cloider_domain_verification" "example" {
  domain = cloider_domain.example.name

  depends_on = [aws_route53_record.cloider_verify]
}

# Taking the domain from the verification creates the target once it's verified
resource "cloider_target" "web" {
  domain     = cloider_domain_verification.example.domain
  cluster_id = cloider_cluster.web.id
  namespace  = "shop"
  service    = "web"
  port       = 80
}
