# Multiple environments

A staging and a production environment from one module, each with a Kubernetes cluster, a PostgreSQL
database only the cluster (and, for staging, an office network) can reach, a versioned bucket and a
key for it. The environments differ only in what the module is given: production gets a larger
database plan.

Two environments need more than a free account's room for one cluster, one database and one bucket.

## Usage

```sh
export CLOIDER_TOKEN="cldr_..."
terraform init
terraform apply -var bucket_prefix=acme
```

Pause staging when nobody uses it, which keeps its state and stops billing its compute, by giving its
module `paused = true` and applying again.

To keep the environments' states apart, so a mistake in one plan can't touch the other, give each
environment its own folder, or its own workspace, calling the same module.
