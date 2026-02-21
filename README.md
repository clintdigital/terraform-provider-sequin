# Terraform Provider for Sequin

Terraform provider for managing [Sequin](https://sequinstream.com) resources: databases, sink consumers, and backfills.

## Quick Start

```hcl
terraform {
  required_providers {
    sequin = {
      source  = "clintdigital/sequin"
      version = "~> 0.1"
    }
  }
}

provider "sequin" {
  endpoint = "https://your-instance.sequin.io"
  api_key  = var.sequin_api_key
}
```

Or use environment variables: `SEQUIN_ENDPOINT` and `SEQUIN_API_KEY`.

---

## Provider Configuration

| Argument   | Type   | Required | Description |
|------------|--------|----------|-------------|
| `endpoint` | string | Yes      | Sequin API endpoint URL. Also `SEQUIN_ENDPOINT` env var. |
| `api_key`  | string | Yes      | API authentication key. Also `SEQUIN_API_KEY` env var. Sensitive. |

---

## Resources

### `sequin_database`

Connects Sequin to a PostgreSQL database for CDC (Change Data Capture).

```hcl
resource "sequin_database" "main" {
  name     = "production-db"
  hostname = "postgres.example.com"
  port     = 5432
  database = "myapp"
  username = var.db_username
  password = var.db_password
  ssl      = true

  replication_slots = [{
    publication_name = "sequin_pub"
    slot_name        = "sequin_slot"
  }]
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Unique name for the database connection. |
| `url` | string | No | Full PostgreSQL connection URL. Alternative to individual connection fields. Sensitive. |
| `hostname` | string | No | Database server hostname. |
| `port` | number | No | Database server port. Defaults to `5432`. |
| `database` | string | No | Logical database name in PostgreSQL. |
| `username` | string | No | Database authentication username. |
| `password` | string | No | Database authentication password. Sensitive. |
| `ssl` | bool | No | Enable SSL for connection. Defaults to `true`. |
| `ipv6` | bool | No | Use IPv6 for connection. Defaults to `false`. |
| `replication_slots` | list | Yes | Replication slot configuration (see below). |
| `primary` | object | No | Primary database config for replica connections (see below). |

**`replication_slots` block:**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `publication_name` | string | Yes | PostgreSQL publication name. |
| `slot_name` | string | Yes | PostgreSQL replication slot name. |
| `status` | string | No | Slot status: `active`, `disabled`. Computed. |
| `id` | string | — | Computed slot ID. |

**`primary` block** (for replica connections):

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `hostname` | string | Yes | Primary database hostname. |
| `database` | string | Yes | Primary database name. |
| `username` | string | Yes | Primary database username. |
| `password` | string | Yes | Primary database password. Sensitive. |
| `port` | number | No | Primary database port. |
| `ssl` | bool | No | Enable SSL for primary connection. |

#### Read-Only Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Unique database connection ID. |
| `use_local_tunnel` | bool | Whether a local tunnel is being used. |
| `pool_size` | number | Connection pool size. |
| `queue_interval` | number | Queue processing interval. |
| `queue_target` | number | Queue processing target. |

#### Import

```bash
terraform import sequin_database.main <database-id>
```

---

### `sequin_sink_consumer`

Streams database changes to a destination (Kafka, SQS, Kinesis, or Webhook).

```hcl
resource "sequin_sink_consumer" "webhook" {
  name     = "orders-to-webhook"
  database = sequin_database.main.id

  tables = [{ name = "public.orders" }]

  actions = ["insert", "update"]

  destination = {
    type               = "webhook"
    http_endpoint      = "https://api.example.com"
    http_endpoint_path = "/webhook"
  }
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Unique name for the sink consumer. |
| `database` | string | Yes | ID of the database connection to stream from. |
| `status` | string | No | Desired status: `active`, `disabled`, `paused`. Computed if not set. |
| `tables` | list | Yes | Tables to stream changes from (see below). |
| `actions` | list(string) | No | Change actions to capture: `insert`, `update`, `delete`. |
| `destination` | object | Yes | Destination configuration (see below). |
| `source` | object | No | Source filtering configuration (see below). |
| `filter` | string | No | Named filter function to control which rows trigger changes. |
| `transform` | string | No | Named transform function to reshape messages before delivery. |
| `enrichment` | string | No | Named enrichment function that runs a SQL query to add data to messages. |
| `routing` | string | No | Named routing function to dynamically direct messages to destinations. |
| `message_grouping` | bool | No | Enable message grouping for ordered delivery. |
| `batch_size` | number | No | Number of messages to batch together. |
| `max_retry_count` | number | No | Maximum retry attempts for failed deliveries. |
| `load_shedding_policy` | string | No | Overload policy: `pause_on_full`, `discard_on_full`. |
| `timestamp_format` | string | No | Timestamp format: `iso8601`, `unix_microsecond`. |

**`tables` block:**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Table name, schema-qualified (e.g. `public.users`). |
| `group_column_names` | list(string) | No | Columns for message grouping/ordering. |

**`destination` block:**

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `type` | string | Yes | Destination type: `kafka`, `sqs`, `kinesis`, `webhook`. |

*Kafka fields:*

| Argument | Type | Description |
|----------|------|-------------|
| `hosts` | string | Broker hosts (comma-separated). |
| `topic` | string | Kafka topic name. |
| `tls` | bool | Enable TLS for connection. |
| `username` | string | Authentication username. |
| `password` | string | Authentication password. Sensitive. |
| `sasl_mechanism` | string | SASL mechanism: `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`, `AWS_MSK_IAM`. |
| `aws_region` | string | AWS region for MSK IAM authentication. |
| `aws_access_key_id` | string | AWS access key ID for MSK IAM. Sensitive. |
| `aws_secret_access_key` | string | AWS secret access key for MSK IAM. Sensitive. |

*SQS fields:*

| Argument | Type | Description |
|----------|------|-------------|
| `queue_url` | string | SQS queue URL. |
| `region` | string | AWS region. |
| `access_key_id` | string | AWS access key ID. Sensitive. |
| `secret_access_key` | string | AWS secret access key. Sensitive. |
| `is_fifo` | bool | Whether the queue is FIFO. |

*Kinesis fields:*

| Argument | Type | Description |
|----------|------|-------------|
| `stream_arn` | string | Kinesis stream ARN. |
| `region` | string | AWS region. |
| `access_key_id` | string | AWS access key ID. Sensitive. |
| `secret_access_key` | string | AWS secret access key. Sensitive. |

*Webhook fields:*

| Argument | Type | Description |
|----------|------|-------------|
| `http_endpoint` | string | Webhook HTTP endpoint base URL. |
| `http_endpoint_path` | string | Webhook HTTP endpoint path. |
| `batch` | bool | Enable batched delivery for webhooks. |

**`source` block** (optional schema/table filtering):

| Argument | Type | Description |
|----------|------|-------------|
| `include_schemas` | list(string) | Schema names to include. |
| `exclude_schemas` | list(string) | Schema names to exclude. |
| `include_tables` | list(string) | Table names to include. |
| `exclude_tables` | list(string) | Table names to exclude. |

#### Read-Only Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Unique sink consumer ID. |
| `status_info.state` | string | Current state: `active`, `pending`, `failed`, `disabled`. |
| `status_info.created_at` | string | ISO 8601 creation timestamp. |
| `status_info.updated_at` | string | ISO 8601 last update timestamp. |
| `status_info.last_error` | string | Most recent error message. |

#### Import

```bash
terraform import sequin_sink_consumer.webhook <consumer-id>
```

---

### `sequin_backfill`

Backfills historical data through a sink consumer.

```hcl
resource "sequin_backfill" "orders" {
  sink_consumer = sequin_sink_consumer.webhook.name
}

# For multi-table consumers, specify the table
resource "sequin_backfill" "users" {
  sink_consumer = sequin_sink_consumer.kafka.name
  table         = "public.users"
}

# Cancel a running backfill
resource "sequin_backfill" "cancelled" {
  sink_consumer = sequin_sink_consumer.webhook.name
  state         = "cancelled"
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `sink_consumer` | string | Yes | Name or ID of the sink consumer. Forces replacement on change. |
| `table` | string | No | Source table (`schema.table` format). Required if the sink streams from multiple tables. Forces replacement on change. |
| `state` | string | No | Desired state: `active`, `cancelled`. Set to `cancelled` to cancel a running backfill. |

#### Read-Only Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Unique backfill ID. |
| `status.state` | string | Current state: `active`, `completed`, `cancelled`. |
| `status.inserted_at` | string | ISO 8601 creation timestamp. |
| `status.updated_at` | string | ISO 8601 last update timestamp. |
| `status.canceled_at` | string | ISO 8601 cancellation timestamp. |
| `status.completed_at` | string | ISO 8601 completion timestamp. |
| `status.rows_ingested_count` | number | Rows delivered to the sink. |
| `status.rows_initial_count` | number | Total rows targeted for processing. |
| `status.rows_processed_count` | number | Rows examined during backfill. |
| `status.sort_column` | string | Column used for ordering. |

#### Import

```bash
terraform import sequin_backfill.orders <sink_consumer_name>/<backfill_id>
```

---

## Development

```bash
make build      # Build the provider
make install    # Install locally for testing
make test       # Run unit tests
make fmt        # Format code
make lint       # Lint code
make docs       # Generate documentation
```

### Pre-commit hooks

```bash
pre-commit install
pre-commit run --all-files
```

See [testing.md](testing.md) for a step-by-step guide to testing the provider locally.

### Project Structure

```
.
├── main.go                  # Entrypoint
├── internal/
│   ├── provider/            # Provider config
│   ├── client/              # HTTP API client
│   └── resources/           # Resource CRUD implementations
├── examples/
│   ├── provider/            # Provider configuration example
│   └── resources/           # Per-resource examples
└── test-provider/           # Local test configuration
```

---

## Releasing

Tag a version and push — GitHub Actions handles the rest (build, sign, publish to registry):

```bash
git tag v0.1.0
git push origin v0.1.0
```

### First-time setup

1. Create a GPG key: `gpg --full-generate-key` (RSA, 4096 bits)
2. Add GitHub secrets: `GPG_PRIVATE_KEY` and `PASSPHRASE`
3. Sign in to [registry.terraform.io](https://registry.terraform.io/), publish the provider, and add your GPG public key
