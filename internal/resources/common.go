package resources

// ResourceStatus represents computed status attributes common across resources.
// These fields are read-only and populated by the API.
type ResourceStatus struct {
	State     string `tfsdk:"state"`      // Resource state: active, pending, failed, disabled
	CreatedAt string `tfsdk:"created_at"` // ISO 8601 timestamp of creation
	UpdatedAt string `tfsdk:"updated_at"` // ISO 8601 timestamp of last update
	LastError string `tfsdk:"last_error"` // Most recent error message if any
}

// BackfillStatus represents computed status attributes specific to backfill resources.
type BackfillStatus struct {
	State              string `tfsdk:"state"`                // Backfill state: active, completed, cancelled
	InsertedAt         string `tfsdk:"inserted_at"`          // ISO 8601 timestamp of creation
	UpdatedAt          string `tfsdk:"updated_at"`           // ISO 8601 timestamp of last update
	CanceledAt         string `tfsdk:"canceled_at"`          // ISO 8601 timestamp of cancellation
	CompletedAt        string `tfsdk:"completed_at"`         // ISO 8601 timestamp of completion
	RowsIngestedCount  int    `tfsdk:"rows_ingested_count"`  // Rows delivered to the sink
	RowsInitialCount   int    `tfsdk:"rows_initial_count"`   // Total rows targeted
	RowsProcessedCount int    `tfsdk:"rows_processed_count"` // Rows examined
	SortColumn         string `tfsdk:"sort_column"`          // Column used for ordering
}
