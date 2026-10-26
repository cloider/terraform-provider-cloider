---
page_title: "cloider_database Resource - cloider"
subcategory: "Databases"
description: |-
  Manages a PostgreSQL, MySQL, MongoDB or Redis database.
---

# Resource: cloider_database

Manages a PostgreSQL, MySQL, MongoDB or Redis database. Creating one waits until it is `Ready`.

Clients connect at `host` from anywhere `trusted_sources` allows, and at `private_host` from your
clusters and apps, inside Cloider's network. With `ssl` on, the server's certificate is signed by
Cloider's own CA, `ca_certificate`, so a client that verifies certificates needs it.

~> **Note:** `name`, `engine`, `version` and `user` can't change, and a plan can only be resized up,
so changing those replaces the database and its data. Consider `prevent_destroy`, as in the example
below.

~> **Note:** The password is kept in Terraform's state, so treat the state as sensitive.

## Example Usage

### Basic Usage

```terraform
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "starter"
}
```

### Restricting Who May Connect

`trusted_sources` defaults to `["*"]`, anywhere. List your clusters, apps and networks instead:

```terraform
resource "cloider_cluster" "web" {
  name = "web"
}

resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"

  # The cluster, the office network and one CI runner
  trusted_sources = [
    "cluster:${cloider_cluster.web.id}",
    "203.0.113.0/24",
    "198.51.100.7/32",
  ]
}
```

### MySQL, MongoDB and Redis

```terraform
resource "cloider_database" "accounts" {
  name   = "accounts"
  engine = "mysql"
}

resource "cloider_database" "events" {
  name   = "events"
  engine = "mongodb"
}

resource "cloider_database" "cache" {
  name   = "cache"
  engine = "redis"
}
```

### Connection Strings

```terraform
# The connection strings the dashboard shows, for clients outside Cloider.
# Use private_host instead of host from your clusters and apps.
locals {
  pg    = cloider_database.orders
  mysql = cloider_database.accounts
  mongo = cloider_database.events
  redis = cloider_database.cache
}

output "postgresql_url" {
  value     = "postgresql://${local.pg.user}:${urlencode(local.pg.password)}@${local.pg.host}:${local.pg.port}/${local.pg.name}?sslmode=require"
  sensitive = true
}

output "mysql_url" {
  value     = "mysql://${local.mysql.user}:${urlencode(local.mysql.password)}@${local.mysql.host}:${local.mysql.port}/${local.mysql.name}"
  sensitive = true
}

output "mongodb_url" {
  value     = "mongodb://${local.mongo.user}:${urlencode(local.mongo.password)}@${local.mongo.host}:${local.mongo.port}/${local.mongo.name}?directConnection=true&tls=true"
  sensitive = true
}

# Redis is TLS only unless ssl = false, then the scheme is redis://
output "redis_url" {
  value     = "rediss://${local.redis.user}:${urlencode(local.redis.password)}@${local.redis.host}:${local.redis.port}/0"
  sensitive = true
}
```

### Passing Credentials to a Kubernetes Workload

```terraform
resource "cloider_database" "orders" {
  name            = "orders"
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}

# The credentials as a Secret in the cluster, with the kubernetes provider
# configured as in the cloider_cluster docs
resource "kubernetes_secret_v1" "orders_db" {
  metadata {
    name      = "orders-db"
    namespace = "default"
  }

  data = {
    DATABASE_URL = "postgresql://${cloider_database.orders.user}:${urlencode(cloider_database.orders.password)}@${cloider_database.orders.private_host}:${cloider_database.orders.port}/${cloider_database.orders.name}?sslmode=require"

    # For clients that verify the server's certificate
    "ca.crt" = cloider_database.orders.ca_certificate
  }
}
```

### Resizing

```terraform
# Moving to a larger plan resizes the database in place, with a restart. A
# smaller plan can't be resized to, so it replaces the database and its data.
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "standard"
}
```

### Protecting a Production Database

```terraform
# Refuses any plan that would delete the database or replace it, such as a
# smaller plan or another version
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "standard"

  lifecycle {
    prevent_destroy = true
  }
}
```

### Custom User and Label

```terraform
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  user   = "orders_app"
  label  = "orders-production"
}
```

### Redis without TLS

```terraform
# Redis is TLS only by default. ssl = false serves plain redis:// instead,
# for clients without TLS support. Keep its trusted sources tight.
resource "cloider_database" "cache" {
  name            = "cache"
  engine          = "redis"
  ssl             = false
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}
```

### Pausing a Database

```terraform
# A paused database keeps its data and can't be connected to
resource "cloider_database" "staging" {
  name   = "staging"
  engine = "postgresql"
  paused = true
}
```

### Timeouts

```terraform
resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = "pro"

  timeouts = {
    create = "40m"
    update = "40m"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `engine` (String) `postgresql`, `mysql`, `mongodb` or `redis`.
- `name` (String) The database's name: lower case letters, digits and `-`, starting with a letter.

### Optional

- `label` (String) The name the dashboard shows: lower case letters, digits and `-`, starting with a letter. Defaults to `name`, with a `-1`, `-2`... suffix when another database has it. It has to be unique among your databases.
- `paused` (Boolean) Whether the database is paused. A paused database keeps its data and can't be connected to. Defaults to `false`.
- `plan` (String) `hobby`, `starter`, `basic`, `standard`, `plus` or `pro`, see `cloider pricing`. Moving to a larger plan resizes the database in place, with a restart. A smaller one replaces it. Defaults to `hobby`.
- `ssl` (Boolean) Whether connections must use TLS. Changing it restarts the database. Defaults to on, except for Redis, where it makes the server TLS only.
- `timeouts` (Attributes) (see [below for nested schema](#nestedatt--timeouts))
- `trusted_sources` (Set of String) Who may connect: `*` for anywhere, an IPv4 CIDR such as `203.0.113.0/24` (`/32` for one address), `cluster:<id>` for one of your clusters or `app:<id>` for one of your apps. Defaults to `["*"]`, as the dashboard does.
- `user` (String) The user to connect as: lower case letters, digits and `_`. Defaults to `<name>_user`, with `-` as `_`.
- `version` (String) The engine version, from `cloider_options`' `database_engines`. Defaults to the newest offered.

### Read-Only

- `ca_certificate` (String, Sensitive) The PEM CA certificate that signs the database's TLS certificate, when `ssl` is on.
- `created_at` (String) When the database was created, RFC 3339.
- `host` (String) The host to connect to from anywhere the trusted sources allow.
- `id` (String) The database's id.
- `password` (String, Sensitive) The password of `user`.
- `port` (Number) The port to connect to.
- `private_host` (String) The host to connect to from your clusters and apps, inside Cloider's network.
- `status` (String) `Creating`, `Starting`, `Ready`, `Upgrading`, `Pausing`, `Paused` or `Terminating`.

<a id="nestedatt--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
- `delete` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Setting a timeout for a Delete operation is only applicable if changes are saved into state before the destroy operation occurs.
- `update` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).

## Timeouts

[Configuration options](https://developer.hashicorp.com/terraform/language/resources/syntax#operation-timeouts):

* `create` - (Default `20m`) Waiting for the database to be `Ready`.
* `update` - (Default `20m`) Waiting for a resize, an SSL change, a pause or a resume.
* `delete` - (Default `20m`)

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import)
to import a database by its id. `cloider db list -o wide` shows the ids:

```terraform
import {
  to = cloider_database.orders
  id = "p0q9zm"
}
```

Using [`terraform import`](https://developer.hashicorp.com/terraform/cli/commands/import):

```shell
# A database is imported by its id, see `cloider db list -o wide`
terraform import cloider_database.orders p0q9zm
```
