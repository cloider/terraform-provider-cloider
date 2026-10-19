# DNS for a custom domain

Serves a domain you own from Cloider's DNS: the domain and `www` pointing at a web server, mail
records (MX, SPF and DMARC), and any other records from a map.

## Usage

```sh
cp terraform.tfvars.example terraform.tfvars   # then edit it
export CLOIDER_TOKEN="cldr_..."
terraform init
terraform apply
```

Then set the domain's nameservers at its registrar to the `nameservers` output. The records are
answered once the registrar's change has spread, which can take up to a day or two.
