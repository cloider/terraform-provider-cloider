# Examples

Complete configurations to start from. Each folder is its own root module: `cd` into it,
`terraform init` and `terraform apply`. They take the API token from `CLOIDER_TOKEN`, or from
`cloider login` on your own machine.

| Example                                            | What it creates                                                                                       |
|----------------------------------------------------|-------------------------------------------------------------------------------------------------------|
| [getting-started](getting-started)                 | A cluster, a PostgreSQL database only it can reach, and a bucket with a key. Fits a free account.     |
| [kubernetes-app](kubernetes-app)                   | An app on its own cluster: registry project and tokens, database, the Kubernetes objects to run it, and a domain serving it. |
| [custom-domain](custom-domain)                     | DNS for a domain you own: web, mail (MX, SPF, DMARC) and any other records.                           |
| [multiple-environments](multiple-environments)     | Staging and production from one module, each with a cluster, a database and a bucket.                 |

The other folders hold the snippets the provider's documentation shows: `provider/` for the provider
page, `resources/` and `data-sources/` for each resource and data source, and `guides/` for the
guides.
