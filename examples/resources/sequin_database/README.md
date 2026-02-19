# sequin_database

Manages a PostgreSQL database connection in Sequin for CDC.

## Usage

```hcl
resource "sequin_database" "main" {
  name     = "production-db"
  hostname = "db.example.com"
  port     = 5432
  database = "myapp"
  username = "sequin_user"
  password = var.db_password
  ssl      = true

  replication_slots = [{
    publication_name = "sequin_pub"
    slot_name        = "sequin_slot"
  }]
}
```

### Using a connection URL

```hcl
resource "sequin_database" "main" {
  name = "production-db"
  url  = "postgresql://user:pass@host:5432/db?sslmode=require"

  replication_slots = [{
    publication_name = "sequin_pub"
    slot_name        = "sequin_slot"
  }]
}
```

## Inputs

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | `string` | yes | Unique name for the connection |
| `url` | `string` | no | Full connection URL (alternative to individual fields). Sensitive |
| `hostname` | `string` | no | Database host. Required if `url` not set |
| `port` | `number` | no | Database port. Default: `5432` |
| `database` | `string` | no | Database name. Required if `url` not set |
| `username` | `string` | no | Username. Required if `url` not set |
| `password` | `string` | no | Password. Sensitive. Required if `url` not set |
| `ssl` | `bool` | no | Enable SSL. Default: `true` |
| `ipv6` | `bool` | no | Use IPv6. Default: `false` |
| `replication_slots` | `list(object)` | yes | See below |
| `primary` | `object` | no | Primary DB config for replica connections. See below |

### `replication_slots`

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `publication_name` | `string` | yes | PostgreSQL publication name |
| `slot_name` | `string` | yes | PostgreSQL replication slot name |
| `status` | `string` | no | `active` or `disabled`. Computed |

### `primary`

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `hostname` | `string` | yes | Primary DB host |
| `database` | `string` | yes | Primary DB name |
| `username` | `string` | yes | Primary DB user |
| `password` | `string` | yes | Primary DB password. Sensitive |
| `port` | `number` | no | Primary DB port |
| `ssl` | `bool` | no | Enable SSL for primary |

## Outputs

| Name | Description |
|------|-------------|
| `id` | Database connection ID |
| `use_local_tunnel` | Whether a local tunnel is used |
| `pool_size` | Connection pool size |

## Import

```bash
terraform import sequin_database.main <database-id>
```
