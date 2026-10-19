# App on a Cloider cluster

A containerized app with a PostgreSQL database, on its own Kubernetes cluster:

* a `cloider_cluster`, and a `cloider_database` only the cluster can reach
* a `cloider_registry` project, with a token for CI to push images and a read-only one for the cluster to pull them
* in the cluster: a namespace, an image pull secret, a `DATABASE_URL` secret, a deployment and a service
* a free `cloider.app` subdomain, and a `cloider_target` that serves the service on it over HTTPS

It fits in a free Cloider account.

## Usage

The kubernetes provider is configured from the cluster, so create the cluster first, then the rest:

```sh
export CLOIDER_TOKEN="cldr_..."
terraform init
terraform apply -target=cloider_cluster.app -var registry_project=acme-shop -var subdomain=acme-shop
terraform apply -var registry_project=acme-shop -var subdomain=acme-shop
```

Push the app's image with the CI token. The app listens on port 8080 and reads `DATABASE_URL`:

```sh
terraform output -raw registry_push_password |
  docker login registry.cloider.com -u "$(terraform output -raw registry_push_username)" --password-stdin
docker build -t "$(terraform output -raw registry_url)/shop:latest" .
docker push "$(terraform output -raw registry_url)/shop:latest"
```

The pods start once the image is there, and the app answers at the `url` output,
`https://acme-shop.cloider.app`.

Delete it all with `terraform destroy -var registry_project=acme-shop -var subdomain=acme-shop`.
