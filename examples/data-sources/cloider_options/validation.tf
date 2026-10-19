data "cloider_options" "current" {}

variable "database_plan" {
  type    = string
  default = "starter"
}

resource "cloider_database" "orders" {
  name   = "orders"
  engine = "postgresql"
  plan   = var.database_plan

  lifecycle {
    precondition {
      condition     = contains(data.cloider_options.current.database_plans, var.database_plan)
      error_message = "database_plan has to be one of ${join(", ", data.cloider_options.current.database_plans)}."
    }
  }
}
