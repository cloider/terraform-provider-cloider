---
page_title: "Using Buckets with S3 Tools"
subcategory: ""
description: |-
  Reach Cloider buckets with the AWS CLI, AWS SDKs, rclone and the AWS provider.
---

# Using Buckets with S3 Tools

A [`cloider_bucket`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/bucket)
speaks the S3 api, so the AWS CLI, every AWS SDK, rclone and the
[`aws`](https://registry.terraform.io/providers/hashicorp/aws/latest/docs) provider work with it.
They need four settings:

| Setting    | Value                                                                    |
|------------|--------------------------------------------------------------------------|
| Endpoint   | The bucket's `endpoint`, such as `https://s3.eu-central-2.cloider.com`   |
| Region     | The bucket's `region`, such as `eu-central-2`                            |
| Access key | A `cloider_bucket_key`'s `access_key`                                    |
| Secret key | The same key's `secret_key`                                              |

The regions and their endpoints are listed by
[`cloider_bucket_regions`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/data-sources/bucket_regions).

## Create a Bucket and a Key

```terraform
resource "cloider_bucket" "uploads" {
  name = "acme-uploads"
}

resource "cloider_bucket_key" "app" {
  label   = "app"
  buckets = [cloider_bucket.uploads.id]
}

# The settings any S3 client takes
output "s3_environment" {
  value = {
    AWS_ACCESS_KEY_ID   = cloider_bucket_key.app.access_key
    AWS_REGION          = cloider_bucket.uploads.region
    AWS_ENDPOINT_URL_S3 = cloider_bucket.uploads.endpoint
    S3_BUCKET           = cloider_bucket.uploads.name
  }
}
```

Output the secret key too, to hand it to the tools below:

```terraform
output "s3_access_key" {
  value = cloider_bucket_key.app.access_key
}

output "s3_secret_key" {
  value     = cloider_bucket_key.app.secret_key
  sensitive = true
}
```

~> **Note:** The secret key is only shown when the key is created, so it can't be read back later. Keep
it in Terraform's state, or recreate the key.

## AWS CLI

A `cloider` profile, as the dashboard shows it:

```shell
aws configure set aws_access_key_id "$(terraform output -raw s3_access_key)" --profile cloider
aws configure set aws_secret_access_key "$(terraform output -raw s3_secret_key)" --profile cloider
aws configure set region eu-central-2 --profile cloider
aws configure set endpoint_url https://s3.eu-central-2.cloider.com --profile cloider

aws s3 cp ./photo.jpg s3://acme-uploads/ --profile cloider
aws s3 ls s3://acme-uploads --profile cloider
```

## AWS SDKs and Other Clients

The AWS SDKs read these environment variables, `AWS_ENDPOINT_URL_S3` included:

```shell
export AWS_ACCESS_KEY_ID="$(terraform output -raw s3_access_key)"
export AWS_SECRET_ACCESS_KEY="$(terraform output -raw s3_secret_key)"
export AWS_REGION=eu-central-2
export AWS_ENDPOINT_URL_S3=https://s3.eu-central-2.cloider.com
```

## rclone

```ini
[cloider]
type = s3
provider = Other
access_key_id = <access key>
secret_access_key = <secret key>
endpoint = https://s3.eu-central-2.cloider.com
region = eu-central-2
```

## Uploading Files with the AWS Provider

Keep the files in a configuration of their own, with the key in its environment, rather than in the
one that creates the key:

```terraform
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
```

## Passing a Key to a Kubernetes Workload

Hand the same settings to an app in a Cloider cluster as a Secret, as the
[`cloider_bucket_key`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/bucket_key#passing-the-key-to-a-kubernetes-workload)
examples show.

## Terraform State

A bucket can hold Terraform's own state too, see
[Storing Terraform State in a Bucket](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/state-backend).

## Public Files

A bucket with `public = true` serves each file without a key at `<url>/<key>`, where `url` is the
bucket's, such as `https://acme-assets.s3.eu-central-2.cloider.com/logo.png`. Listing the bucket and
changing its files still need a key.
