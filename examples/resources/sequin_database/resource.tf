resource "sequin_database" "basic" {
  name     = "production-postgres"
  hostname = "postgres.example.com"
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

resource "sequin_database" "url" {
  name = "staging-postgres"
  url  = "postgresql://user:pass@host:5432/db?sslmode=require"

  replication_slots = [{
    publication_name = "sequin_pub"
    slot_name        = "sequin_slot"
  }]
}

resource "sequin_database" "replica" {
  name     = "analytics-replica"
  hostname = "postgres-replica.example.com"
  port     = 5432
  database = "myapp"
  username = "sequin_user"
  password = var.replica_password
  ssl      = true

  primary = {
    hostname = "postgres-primary.example.com"
    database = "myapp"
    username = "sequin_user"
    password = var.primary_password
    port     = 5432
    ssl      = true
  }

  replication_slots = [{
    publication_name = "sequin_pub"
    slot_name        = "sequin_replica_slot"
  }]
}
