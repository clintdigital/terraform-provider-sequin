# sequin_sink_consumer

Streams database changes (CDC) to Kafka, SQS, Kinesis, or Webhook endpoints.

## Usage

### Kafka

```hcl
resource "sequin_sink_consumer" "events" {
  name     = "events-to-kafka"
  database = sequin_database.main.id

  tables  = [{ name = "public.events" }]
  actions = ["insert", "update", "delete"]

  destination = {
    type           = "kafka"
    hosts          = "broker1:9092,broker2:9092"
    topic          = "database.events"
    tls            = true
    username       = "user"
    password       = var.kafka_pass
    sasl_mechanism = "scram_sha_256"
  }
}
```

### SQS

```hcl
destination = {
  type              = "sqs"
  queue_url         = "https://sqs.us-east-1.amazonaws.com/123/queue"
  region            = "us-east-1"
  access_key_id     = var.aws_key
  secret_access_key = var.aws_secret
}
```

### Kinesis

```hcl
destination = {
  type              = "kinesis"
  stream_arn        = "arn:aws:kinesis:us-east-1:123:stream/events"
  region            = "us-east-1"
  access_key_id     = var.aws_key
  secret_access_key = var.aws_secret
}
```

### Webhook

```hcl
destination = {
  type               = "webhook"
  http_endpoint      = "https://api.example.com"
  http_endpoint_path = "/webhooks/events"
}
```

## Inputs

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | `string` | yes | Consumer name |
| `database` | `string` | yes | Database ID or name |
| `tables` | `list(object)` | yes | Tables to stream. `[{ name, group_column_names? }]` |
| `destination` | `object` | yes | Destination config. See below |
| `status` | `string` | no | `active`, `disabled`, `paused`. Computed |
| `source` | `object` | no | Schema/table filtering. See below |
| `actions` | `list(string)` | no | `insert`, `update`, `delete` |
| `filter` | `string` | no | Named filter function |
| `transform` | `string` | no | Named transform function |
| `enrichment` | `string` | no | Named enrichment function |
| `routing` | `string` | no | Named routing function |
| `batch_size` | `number` | no | Messages per batch. Computed |
| `message_grouping` | `bool` | no | Ordered delivery. Computed |
| `max_retry_count` | `number` | no | Max retries |
| `load_shedding_policy` | `string` | no | `pause_on_full`, `discard_on_full`. Computed |
| `timestamp_format` | `string` | no | `iso8601`, `unix_microsecond`. Computed |

### `destination`

All destinations require `type` (`kafka`, `sqs`, `kinesis`, `webhook`).

| Field | Kafka | SQS | Kinesis | Webhook |
|-------|:-----:|:---:|:-------:|:-------:|
| `hosts` | **required** | | | |
| `topic` | **required** | | | |
| `tls` | optional | | | |
| `username` | optional | | | |
| `password` | optional | | | |
| `sasl_mechanism` | optional | | | |
| `aws_region` | optional* | | | |
| `aws_access_key_id` | optional* | | | |
| `aws_secret_access_key` | optional* | | | |
| `queue_url` | | **required** | | |
| `region` | | **required** | **required** | |
| `access_key_id` | | **required** | **required** | |
| `secret_access_key` | | **required** | **required** | |
| `is_fifo` | | optional | | |
| `stream_arn` | | | **required** | |
| `http_endpoint` | | | | **required** |
| `http_endpoint_path` | | | | optional |
| `batch` | | | | optional |

*Required when `sasl_mechanism = "aws_msk_iam"`

### `source`

| Name | Type | Description |
|------|------|-------------|
| `include_schemas` | `list(string)` | Only these schemas |
| `exclude_schemas` | `list(string)` | All except these |
| `include_tables` | `list(string)` | Only these tables |
| `exclude_tables` | `list(string)` | All except these |

## Outputs

| Name | Description |
|------|-------------|
| `id` | Consumer ID |
| `status_info` | Operational status (`state`, `created_at`, `updated_at`, `last_error`) |

## Import

```bash
terraform import sequin_sink_consumer.events <consumer-id>
```
