# sequin_backfill

Replays historical rows through a sink consumer's pipeline.

## Usage

```hcl
resource "sequin_backfill" "orders" {
  sink_consumer = sequin_sink_consumer.orders.name
}
```

### Specific table (multi-table consumer)

```hcl
resource "sequin_backfill" "users" {
  sink_consumer = sequin_sink_consumer.multi.name
  table         = "public.users"
}
```

### Cancel a backfill

```hcl
resource "sequin_backfill" "orders" {
  sink_consumer = sequin_sink_consumer.orders.name
  state         = "cancelled"
}
```

## Inputs

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sink_consumer` | `string` | yes | Consumer name or ID. Forces replacement |
| `table` | `string` | no | `schema.table` format. Required if consumer has multiple tables. Forces replacement |
| `state` | `string` | no | `active` or `cancelled`. Computed |

## Outputs

| Name | Description |
|------|-------------|
| `id` | Backfill ID |
| `status` | Status object (`state`, `inserted_at`, `updated_at`, `canceled_at`, `completed_at`, `rows_ingested_count`, `rows_initial_count`, `rows_processed_count`, `sort_column`) |

## Import

```bash
terraform import sequin_backfill.orders <consumer-name>/<backfill-id>
```
