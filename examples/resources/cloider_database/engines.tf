resource "cloider_database" "accounts" {
  name   = "accounts"
  engine = "mysql"
}

resource "cloider_database" "events" {
  name   = "events"
  engine = "mongodb"
}

resource "cloider_database" "cache" {
  name   = "cache"
  engine = "redis"
}
