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

Cloider only serves the records of a domain you've verified, and deletes an unverified one 7 days
after it was added:

1. Add the TXT record from the `verification_record` output where the domain's DNS is hosted now.
2. Set `verify = true` in `terraform.tfvars` and apply again, which waits until Cloider finds the
   record.
3. Set the domain's nameservers at its registrar to the `nameservers` output. The records are
   answered once the registrar's change has spread, which can take up to a day or two.

Verify the domain before moving its nameservers: Cloider's nameservers don't answer for an
unverified domain, so its TXT record can't be found there. A domain whose record was already in
place, or one under a domain you verified, is verified as soon as it's added, with no record to add.
