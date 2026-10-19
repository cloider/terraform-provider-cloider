# Getting started

A Kubernetes cluster, a PostgreSQL database only the cluster can reach, and a bucket with an access
key for it. It fits in a free Cloider account. The
[Getting Started guide](https://registry.terraform.io/providers/cloider/cloider/latest/docs/guides/getting-started)
walks through it.

## Usage

```sh
export CLOIDER_TOKEN="cldr_..."   # or run `cloider login`
terraform init
terraform apply -var prefix=acme

terraform output -raw kubeconfig > kubeconfig.yaml
KUBECONFIG=./kubeconfig.yaml kubectl get namespaces
```

Delete it all with `terraform destroy -var prefix=acme`.
