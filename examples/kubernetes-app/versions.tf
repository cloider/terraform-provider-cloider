terraform {
  required_version = ">= 1.5"

  required_providers {
    cloider = {
      source  = "cloider/cloider"
      version = "~> 0.1"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 3.0"
    }
  }
}
