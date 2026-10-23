# Served as a static website at website_url: / and every path ending in a
# slash get index.html, anything missing gets 404.html with a 404. The website
# is read without a key, so the bucket has to be public.
resource "cloider_bucket" "site" {
  name           = "acme-site"
  public         = true
  website        = true
  index_document = "index.html"
  error_document = "404.html"
}

output "site_url" {
  value = cloider_bucket.site.website_url
}
