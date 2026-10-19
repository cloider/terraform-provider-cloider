locals {
  dns_records = {
    www = "1042"
    api = "1043"
  }
}

import {
  for_each = local.dns_records
  to       = cloider_dns_record.this[each.key]
  id       = each.value
}
