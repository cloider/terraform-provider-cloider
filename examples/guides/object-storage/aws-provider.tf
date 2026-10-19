# A separate configuration, with the key in AWS_ACCESS_KEY_ID and
# AWS_SECRET_ACCESS_KEY. The skip_ settings stop the provider from calling
# AWS's own account and metadata apis.
provider "aws" {
  region = "eu-central-2"

  skip_credentials_validation = true
  skip_requesting_account_id  = true
  skip_metadata_api_check     = true

  endpoints {
    s3 = "https://s3.eu-central-2.cloider.com"
  }
}

resource "aws_s3_object" "robots" {
  bucket       = "acme-assets"
  key          = "robots.txt"
  content      = "User-agent: *\nDisallow:\n"
  content_type = "text/plain"
}

# Uploads every file under ./site, again when it changes
locals {
  content_types = {
    html = "text/html"
    css  = "text/css"
    js   = "text/javascript"
    png  = "image/png"
    svg  = "image/svg+xml"
  }
}

resource "aws_s3_object" "site" {
  for_each = fileset("${path.module}/site", "**")

  bucket       = "acme-assets"
  key          = each.value
  source       = "${path.module}/site/${each.value}"
  source_hash  = filemd5("${path.module}/site/${each.value}")
  content_type = lookup(local.content_types, reverse(split(".", each.value))[0], "application/octet-stream")
}
