export AWS_ACCESS_KEY_ID="$(terraform output -raw s3_access_key)"
export AWS_SECRET_ACCESS_KEY="$(terraform output -raw s3_secret_key)"
export AWS_REGION=eu-central-2
export AWS_ENDPOINT_URL_S3=https://s3.eu-central-2.cloider.com
