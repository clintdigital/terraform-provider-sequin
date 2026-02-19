# Sequin CDC Module

Provisions a complete Sequin CDC pipeline: database connection + sink consumers (Kafka, SQS, Kinesis, Webhook) + optional backfills.

## Usage

```hcl
module "sequin" {
  source = "clintdigital/sequin"

  database_name = "prod-db"
  postgres_host = "db.example.com"
  postgres_db   = "myapp"
  postgres_user = "sequin"
  postgres_pass = var.db_password

  consumers = {
    orders-to-kafka = {
      tables = {
        include = [{ name = "public.orders" }]
      }

      destination = {
        type           = "kafka"
        hosts          = "broker1:9092,broker2:9092"
        topic          = "database.orders"
        tls            = true
        username       = "user"
        password       = var.kafka_pass
        sasl_mechanism = "scram_sha_256"
      }
    }
  }
}
```

### Schema and table filtering

```hcl
consumers = {
  # All schemas, all tables (default — just omit both)
  everything = {
    destination = { ... }
  }

  # Only specific tables
  specific-tables = {
    tables = {
      include = [
        { name = "public.orders", group_column_names = ["customer_id"] },
        { name = "public.order_items" }
      ]
    }
    destination = { ... }
  }

  # Only the "public" schema
  public-only = {
    schemas = { include = ["public"] }
    tables  = { include = [{ name = "public.orders" }] }
    destination = { ... }
  }

  # All schemas except "internal"
  skip-internal = {
    schemas = { exclude = ["internal"] }
    tables  = { include = [{ name = "public.events" }] }
    destination = { ... }
  }

  # Exclude specific tables
  skip-audit = {
    tables = {
      include = [{ name = "public.orders" }]
      exclude = [{ name = "public.audit_log" }]
    }
    destination = { ... }
  }
}
```

### Multiple consumers

```hcl
consumers = {
  orders-to-kafka = {
    schemas = { include = ["public"] }
    tables  = {
      include = [
        { name = "public.orders", group_column_names = ["customer_id"] },
        { name = "public.order_items" }
      ]
    }
    actions          = ["insert", "update"]
    filter_function  = "orders-filter"
    routing_function = "topic-router"

    destination = {
      type           = "kafka"
      hosts          = "broker1:9092"
      topic          = "database.orders"
      tls            = true
      username       = "user"
      password       = var.kafka_pass
      sasl_mechanism = "scram_sha_256"
    }

    batch_size       = 10
    message_grouping = true
  }

  events-to-sqs = {
    tables = { include = [{ name = "public.events" }] }

    destination = {
      type              = "sqs"
      queue_url         = "https://sqs.us-east-1.amazonaws.com/123/events"
      region            = "us-east-1"
      access_key_id     = var.aws_key
      secret_access_key = var.aws_secret
    }
  }

  notifications-to-webhook = {
    tables = { include = [{ name = "public.notifications" }] }

    destination = {
      type               = "webhook"
      http_endpoint      = "https://api.example.com"
      http_endpoint_path = "/webhooks"
    }
  }
}
```

### Per-user provisioning with `for_each`

```hcl
variable "prefixes" {
  default = ["igor", "john", "maria"]
}

module "sequin" {
  source   = "clintdigital/sequin"
  for_each = toset(var.prefixes)

  database_name = "${each.key}_database"
  postgres_host = "db.example.com"
  postgres_db   = each.key
  postgres_user = "sequin"
  postgres_pass = var.db_password

  replication_slots = [{
    publication_name = "${each.key}_pub"
    slot_name        = "${each.key}_slot"
  }]

  consumers = {
    "${each.key}_sink" = {
      tables = { include = [{ name = "public.tags" }] }
      destination = {
        type  = "kafka"
        hosts = "broker1:9092"
        topic = "${each.key}.events"
      }
    }
  }
}
```

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| `database_name` | Database connection name | `string` | — | yes |
| `postgres_host` | PostgreSQL host | `string` | — | yes |
| `postgres_port` | PostgreSQL port | `number` | `5432` | no |
| `postgres_db` | PostgreSQL database | `string` | — | yes |
| `postgres_user` | PostgreSQL username | `string` | — | yes |
| `postgres_pass` | PostgreSQL password | `string` | — | yes |
| `postgres_ssl` | Enable SSL | `bool` | `true` | no |
| `replication_slots` | Replication slot config | `list(object)` | `[{sequin_pub, sequin_slot}]` | no |
| `consumers` | Sink consumers to create | `map(object)` | `{}` | no |
| `backfills` | Backfills to create | `map(object)` | `{}` | no |
| `prevent_destroy` | Prevent database destruction | `bool` | `true` | no |

### Consumer object

| Field | Description | Default |
|-------|-------------|---------|
| `schemas` | `{ include = [...] }` or `{ exclude = [...] }`. Omit for all. | all |
| `tables` | `{ include = [{ name, group_column_names? }] }` and/or `{ exclude = [{ name }] }`. Omit for all. | all |
| `actions` | `["insert", "update", "delete"]` | all three |
| `filter_function` | Named filter function | `null` |
| `enrichment_function` | Named enrichment function | `null` |
| `transform_function` | Named transform function | `null` |
| `routing_function` | Named routing function | `null` |
| `destination` | **Required.** See destination fields below. | — |
| `status` | `active`, `disabled`, `paused` | computed |
| `batch_size` | Messages per batch | computed |
| `message_grouping` | Ordered delivery | computed |
| `max_retry_count` | Max retries | `null` |
| `load_shedding_policy` | `pause_on_full`, `discard_on_full` | computed |
| `timestamp_format` | `iso8601`, `unix_microsecond` | computed |

> **Note:** The `filter_function`, `enrichment_function`, `transform_function`, and `routing_function` fields reference functions by name. These functions must be created in the Sequin UI before they can be referenced in your Terraform configuration.

### Destination fields by type

**Kafka:** `hosts`, `topic`, `tls`, `username`, `password`, `sasl_mechanism`, `aws_region`, `aws_access_key_id`, `aws_secret_access_key`

**SQS:** `queue_url`, `region`, `access_key_id`, `secret_access_key`, `is_fifo`

**Kinesis:** `stream_arn`, `region`, `access_key_id`, `secret_access_key`

**Webhook:** `http_endpoint`, `http_endpoint_path`, `batch`

## Outputs

| Name | Description |
|------|-------------|
| `database_id` | ID of the database connection |
| `consumer_ids` | Map of consumer name → ID |
| `backfill_ids` | Map of backfill name → ID |

## Import

```bash
# Database
terraform import 'module.sequin.sequin_database.this' <database-id>

# Consumer
terraform import 'module.sequin.sequin_sink_consumer.this["consumer-name"]' <consumer-id>

# With for_each
terraform import 'module.sequin["igrzi"].sequin_database.this' <database-id>
```
