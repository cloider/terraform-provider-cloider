# Reads every bucket, those created later included
resource "cloider_bucket_key" "backup_reader" {
  label       = "Backup reader"
  all_buckets = true
  read_only   = true
}
