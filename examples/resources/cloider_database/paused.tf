# A paused database keeps its data and can't be connected to
resource "cloider_database" "staging" {
  name   = "staging"
  engine = "postgresql"
  paused = true
}
