# An app deployed with `cloider app`, by its id from `cloider app list -o wide`
resource "cloider_target" "app" {
  domain = cloider_domain.shop.name
  app_id = "a7c2mq"
}
