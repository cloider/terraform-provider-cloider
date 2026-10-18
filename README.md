# Terraform provider for Cloider

Manages [Cloider](https://cloider.com) clusters, databases, buckets and their keys, container
registry projects and their tokens, domains and DNS records, with Terraform or OpenTofu.

```hcl
terraform {
  required_providers {
    cloider = {
      source = "cloider/cloider"
    }
  }
}

provider "cloider" {}

resource "cloider_cluster" "web" {
  name = "web"
}

resource "cloider_database" "orders" {
  name            = "orders"
  engine          = "postgresql"
  trusted_sources = ["cluster:${cloider_cluster.web.id}"]
}
```

The provider takes its API token from `CLOIDER_TOKEN`, or from `cloider login` on your own machine.
The full reference is on the [Terraform Registry](https://registry.terraform.io/providers/cloider/cloider/latest/docs).

This repository is a mirror. The provider is developed in Cloider's monorepo, next to the api it
talks to, and is pushed here on every release. Issues are welcome here, but pull requests can't be
merged from this side.

## Development

The provider is the `terraform/` folder of the services monorepo, its own Go module with its own
`go.work`, so it builds the way this mirror does and stays out of the services' workspace.

```sh
cd services/terraform
go install .
go test ./...                  # the unit tests, and Terraform against a fake api
go generate ./...              # regenerates docs/ from the schema, examples/ and templates/
```

`docs/` is generated, edit the schema descriptions, `examples/` or `templates/` instead. CI fails
when `docs/` doesn't match.

To run a local build, point Terraform at it in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "cloider/cloider" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

Against local dev, `CLOIDER_API_URL=https://localhost/api` with `CLOIDER_INSECURE=1` to accept the
local proxy's self-signed certificate. `cloider.dev` is behind basic auth, which collides with the
token's Bearer header, so reach the dev api through
`kubectl --context dev -n api port-forward svc/api 18000:8000` and
`CLOIDER_API_URL=http://127.0.0.1:18000/api`.

The acceptance tests create and delete real resources in the token's account, one test after
another, so a free account's room for one cluster, database, bucket and registry project is enough:

```sh
TF_ACC=1 go test ./internal/provider -run TestAcc -timeout 60m
```

The api shapes in `internal/client/types.go` mirror `web/composables/api/**`, as the CLI's do. When
an api route or response the provider uses changes, update it with the web and the CLI.

## Releasing

Push a `tf-v<version>` tag, e.g. `tf-v0.2.0`, on the commit to release. The `terraform-release` job
runs `.gitlab/release-terraform.sh`, which pushes `terraform/` to this mirror's `main` and tags it
`v0.2.0`. That tag runs `.github/workflows/release.yml`: goreleaser builds every platform, signs
the checksums with the provider's GPG key and creates the GitHub release, which the Terraform
Registry picks up. A prerelease (`tf-v0.2.0-rc1`) is published as one, and Terraform only installs
it when a configuration asks for that version.

Set up once:

1. The public GitHub repo `cloider/terraform-provider-cloider`, empty.
2. A GPG key used for nothing else, RSA 4096, with its private half backed up: losing it means
   registering a new key before the next release. The GitHub Actions secret `GPG_PRIVATE_KEY`
   (`gpg --armor --export-secret-keys <id>`), and `PASSPHRASE` when the key has one.
3. A fine-grained GitHub token for that repo with Contents and Workflows read & write, as the
   masked and protected GitLab CI variable `TF_GITHUB_TOKEN`. It expires, renew it before then.
4. On [registry.terraform.io](https://registry.terraform.io), sign in with GitHub, add the GPG
   public key (`gpg --armor --export <id>`) to the `cloider` namespace, then Publish, Provider, and
   pick the repo.
5. For OpenTofu, the "Submit new provider" and "Submit provider signing key" issues on
   [opentofu/registry](https://github.com/opentofu/registry), with your `cloider` org membership public.
