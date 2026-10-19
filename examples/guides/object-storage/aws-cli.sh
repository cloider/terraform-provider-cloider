aws configure set aws_access_key_id "$(terraform output -raw s3_access_key)" --profile cloider
aws configure set aws_secret_access_key "$(terraform output -raw s3_secret_key)" --profile cloider
aws configure set region eu-central-2 --profile cloider
aws configure set endpoint_url https://s3.eu-central-2.cloider.com --profile cloider

aws s3 cp ./photo.jpg s3://acme-uploads/ --profile cloider
aws s3 ls s3://acme-uploads --profile cloider
