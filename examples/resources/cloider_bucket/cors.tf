# Only these websites can use the bucket from a browser: the app uploads with
# presigned URLs and reads the ETag of each part, and the marketing site reads
# files. Without cors_rule blocks any website can.
resource "cloider_bucket" "uploads" {
  name = "acme-uploads"

  cors_rule {
    id              = "web-app"
    allowed_origins = ["https://app.acme.com"]
    allowed_methods = ["GET", "PUT", "POST"]
    allowed_headers = ["*"]
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }

  cors_rule {
    allowed_origins = ["https://*.acme.com"]
    allowed_methods = ["GET", "HEAD"]
  }
}
