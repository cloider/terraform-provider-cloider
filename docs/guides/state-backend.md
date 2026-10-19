---
page_title: "Storing Terraform State in a Bucket"
subcategory: ""
description: |-
  Keep Terraform state in a Cloider bucket with the s3 backend, locking included.
---

# Storing Terraform State in a Bucket

Terraform's [`s3` backend](https://developer.hashicorp.com/terraform/language/backend/s3) works with a
[`cloider_bucket`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/bucket),
locking included, so a team or a CI pipeline can share one state without overwriting each other's
changes.

## Before You Begin

* Terraform 1.10 or later, or OpenTofu 1.10 or later, for `use_lockfile`.

## Create the Bucket and a Key

Create the bucket in a configuration of its own, so the state it holds never depends on itself. A
versioned bucket keeps every earlier state, to go back to one that a mistake overwrote:

```terraform
# A configuration of its own, whose state stays local
resource "cloider_bucket" "state" {
  name       = "acme-terraform-state"
  versioning = true

  lifecycle {
    prevent_destroy = true
  }
}

resource "cloider_bucket_key" "state" {
  label   = "Terraform state"
  buckets = [cloider_bucket.state.id]
}

output "state_access_key" {
  value = cloider_bucket_key.state.access_key
}

output "state_secret_key" {
  value     = cloider_bucket_key.state.secret_key
  sensitive = true
}
```

## Configure the Backend

In each configuration whose state goes in the bucket, give it a `key` of its own:

```terraform
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
```

The backend reads the key from the usual AWS environment variables. Keep them out of the backend
block, which Terraform saves in plain text under `.terraform/`:

```shell
export AWS_ACCESS_KEY_ID="<state_access_key>"
export AWS_SECRET_ACCESS_KEY="<state_secret_key>"
terraform init
```

A configuration that already has a state moves it into the bucket with:

```shell
# Moves a configuration's existing state into the bucket
terraform init -migrate-state
```

## Locking

With `use_lockfile`, a plan or apply first writes `<key>.tflock` next to the state, a write that only
succeeds when no lock file exists. While it holds the lock, any other run against the same state
stops with `Error acquiring the state lock`, or waits as long as `-lock-timeout` allows. The lock file
is removed when the run ends.

~> **Note:** A run that was killed can leave its lock behind. Once you are sure nothing else is
running, release it with `terraform force-unlock <lock id>`, the id the error message shows.
