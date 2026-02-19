# Backfill resource examples
# Backfills process historical data through a sink consumer

# Example 1: Basic backfill for a single-table sink consumer
resource "sequin_backfill" "example" {
  sink_consumer = sequin_sink_consumer.example.name
}

# Example 2: Backfill a specific table (for multi-table sink consumers)
resource "sequin_backfill" "users_backfill" {
  sink_consumer = sequin_sink_consumer.multi_table.name
  table         = "public.users"
}

# Example 3: Cancel a running backfill by setting state
resource "sequin_backfill" "cancellable" {
  sink_consumer = sequin_sink_consumer.example.name
  state         = "cancelled" # Set to "cancelled" to stop the backfill
}
