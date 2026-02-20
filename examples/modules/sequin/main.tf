terraform {
  required_version = ">= 1.0"
  required_providers {
    sequin = {
      source = "clintdigital/sequin"
    }
  }
}

resource "sequin_database" "this" {
  name     = var.database_name
  hostname = var.postgres_host
  port     = var.postgres_port
  database = var.postgres_db
  username = var.postgres_user
  password = var.postgres_pass
  ssl      = var.postgres_ssl

  replication_slots = var.replication_slots
}

resource "sequin_sink_consumer" "this" {
  for_each = var.consumers

  name     = each.key
  database = sequin_database.this.name

  dynamic "source" {
    for_each = each.value.schemas != null || (each.value.tables != null && each.value.tables.exclude != null) ? [1] : []
    content {
      include_schemas = try(each.value.schemas.include, null)
      exclude_schemas = try(each.value.schemas.exclude, null)
      exclude_tables  = try([for t in each.value.tables.exclude : t.name], null)
    }
  }

  tables = each.value.tables != null && each.value.tables.include != null ? each.value.tables.include : []

  actions    = each.value.actions
  filter     = each.value.filter_function
  enrichment = each.value.enrichment_function
  transform  = each.value.transform_function
  routing    = each.value.routing_function

  destination = each.value.destination

  status               = each.value.status
  batch_size           = each.value.batch_size
  message_grouping     = each.value.message_grouping
  max_retry_count      = each.value.max_retry_count
  load_shedding_policy = each.value.load_shedding_policy
  timestamp_format     = each.value.timestamp_format

  depends_on = [sequin_database.this]
}

resource "sequin_backfill" "this" {
  for_each = var.backfills

  sink_consumer = sequin_sink_consumer.this[each.value.consumer_name].name
  table         = each.value.table
  state         = each.value.state

  depends_on = [sequin_sink_consumer.this]
}
