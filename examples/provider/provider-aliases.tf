# One provider block per account, each with that account's token
provider "cloider" {
  profile = "acme"
}

provider "cloider" {
  alias   = "client"
  profile = "client"
}

resource "cloider_bucket" "assets" {
  name = "acme-assets"
}

resource "cloider_bucket" "client_assets" {
  provider = cloider.client

  name = "client-assets"
}
