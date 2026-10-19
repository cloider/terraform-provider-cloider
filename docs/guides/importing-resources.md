---
page_title: "Importing Existing Resources"
subcategory: ""
description: |-
  Bring clusters, databases, buckets and other resources made in the dashboard or the CLI under Terraform.
---

# Importing Existing Resources

Resources created in the dashboard or with the `cloider` CLI can be brought under Terraform without
recreating them. Every Cloider resource can be imported.

## Find the IDs

Most resources are imported by their id. The CLI's `-o wide` output adds an id column:

| Resource                 | Import ID                | Where to find it                         |
|--------------------------|--------------------------|------------------------------------------|
| `cloider_cluster`        | The cluster's id         | `cloider cluster list -o wide`           |
| `cloider_database`       | The database's id        | `cloider db list -o wide`                |
| `cloider_bucket`         | The bucket's id          | `cloider bucket list -o wide`            |
| `cloider_bucket_key`     | The key's id             | `cloider bucket key list -o wide`        |
| `cloider_registry`       | The project's id         | `cloider registry list -o wide`          |
| `cloider_registry_token` | The token's id           | `cloider registry token list -o wide`    |
| `cloider_domain`         | The domain name          | `cloider domain list`                    |
| `cloider_dns_record`     | The record's id          | `cloider dns list <domain> -o wide`      |
| `cloider_target`         | `<domain>/<id>`          | `cloider target list <domain> -o wide`   |

## Import with `import` Blocks

In Terraform v1.5.0 and later, and OpenTofu, write an `import` block for each resource:

```terraform
import {
  to = cloider_cluster.web
  id = "x8f2kq"
}

import {
  to = cloider_database.orders
  id = "p0q9zm"
}

import {
  to = cloider_domain.example
  id = "example.com"
}
```

Terraform can write the matching configuration for you. Review `generated.tf`, move it where it
belongs, then apply to finish the import:

```shell
# Writes the imported resources' configuration to generated.tf
terraform plan -generate-config-out=generated.tf
```

Once applied, the `import` blocks can be deleted.

## Import Many Resources at Once

In Terraform v1.7.0 and later, an `import` block takes `for_each`, to import into a resource that uses
one:

```terraform
locals {
  dns_records = {
    www = "1042"
    api = "1043"
  }
}

import {
  for_each = local.dns_records
  to       = cloider_dns_record.this[each.key]
  id       = each.value
}
```

## Secrets That Stay Unknown

A [`cloider_bucket_key`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/bucket_key)'s
`secret_key` and a [`cloider_registry_token`](https://registry.terraform.io/providers/cloider/cloider/latest/docs/resources/registry_token)'s
`secret` are only shown when they are created, so an imported one has none. When the configuration
needs the secret, to hand it to an app for example, create a new key or token with Terraform instead
of importing the old one, then delete the old one once nothing uses it.

A database's password and a cluster's kubeconfig can be read at any time, so imported ones have them.

## Using the `terraform import` Command

On Terraform versions before v1.5.0, import one resource at a time, after writing its configuration:

```shell
terraform import cloider_cluster.web x8f2kq
```

Each resource's page shows its import command.
