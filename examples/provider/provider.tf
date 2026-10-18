terraform {
  required_providers {
    cloider = {
      source  = "cloider/cloider"
      version = "~> 0.1"
    }
  }
}

# Reads the token from CLOIDER_TOKEN, or the one `cloider login` saved
provider "cloider" {}
