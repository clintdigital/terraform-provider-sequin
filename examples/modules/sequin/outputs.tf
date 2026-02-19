output "database_id" {
  description = "ID of the database connection"
  value       = sequin_database.this.id
}

output "consumer_ids" {
  description = "Map of consumer name → ID"
  value       = { for k, v in sequin_sink_consumer.this : k => v.id }
}

output "backfill_ids" {
  description = "Map of backfill name → ID"
  value       = { for k, v in sequin_backfill.this : k => v.id }
}
