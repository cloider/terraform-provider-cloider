terraform {
  backend "s3" {
    bucket = "acme-terraform-state"
    key    = "production/terraform.tfstate"
    region = "eu-central-2"

    endpoints = {
      s3 = "https://s3.eu-central-2.cloider.com"
    }
    use_path_style = true
    use_lockfile   = true

    # Cloider isn't AWS: there is no AWS account or instance metadata to check
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
  }
}
