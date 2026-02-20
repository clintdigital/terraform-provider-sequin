package resources

import (
	"context"
	"testing"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBackfillResource_Configure(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	// nil provider data
	configResp := &resource.ConfigureResponse{}
	backfillResource.Configure(ctx, resource.ConfigureRequest{ProviderData: nil}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with nil should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// correct client type
	mockClient := &client.Client{}
	configResp = &resource.ConfigureResponse{}
	backfillResource.Configure(ctx, resource.ConfigureRequest{ProviderData: mockClient}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() error: %v", configResp.Diagnostics.Errors())
	}
	if backfillResource.client != mockClient {
		t.Error("Configure() did not set client")
	}
}

func TestBackfillResource_ConfigureWithInvalidType(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	configResp := &resource.ConfigureResponse{}
	backfillResource.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, configResp)
	if !configResp.Diagnostics.HasError() {
		t.Error("Configure() with invalid type should error")
	}
}

func TestBackfillResource_Metadata(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	resp := &resource.MetadataResponse{}
	backfillResource.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sequin"}, resp)

	if resp.TypeName != "sequin_backfill" {
		t.Errorf("TypeName = %q, want sequin_backfill", resp.TypeName)
	}
}

func TestBackfillResource_Schema(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	resp := &resource.SchemaResponse{}
	backfillResource.Schema(ctx, resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() error: %v", resp.Diagnostics.Errors())
	}

	expectedAttrs := []string{"id", "sink_consumer", "table", "state", "status"}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Schema() missing attribute: %s", attr)
		}
	}

	// Verify immutable fields have RequiresReplace modifiers
	for _, field := range []string{"sink_consumer", "table"} {
		if attr, ok := resp.Schema.Attributes[field]; ok {
			if stringAttr, ok := attr.(schema.StringAttribute); ok {
				if len(stringAttr.PlanModifiers) == 0 {
					t.Errorf("field %s should have plan modifiers", field)
				}
			}
		}
	}

	// Verify status nested attribute has all fields
	statusAttr, ok := resp.Schema.Attributes["status"]
	if !ok {
		t.Fatal("missing status attribute")
	}
	nestedAttr, ok := statusAttr.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("status should be SingleNestedAttribute")
	}
	statusFields := []string{"state", "inserted_at", "updated_at", "canceled_at", "completed_at", "rows_ingested_count", "rows_initial_count", "rows_processed_count", "sort_column"}
	for _, field := range statusFields {
		if _, ok := nestedAttr.Attributes[field]; !ok {
			t.Errorf("status missing field: %s", field)
		}
	}
}

// --- mapBackfillResponseToModel tests ---

func TestMapBackfillResponseToModel_BasicMapping(t *testing.T) {
	response := &client.BackfillResponse{
		ID:                 "bf-001",
		State:              "active",
		Table:              "public.users",
		SinkConsumer:       "my-consumer",
		InsertedAt:         "2025-01-15T10:00:00Z",
		UpdatedAt:          "2025-01-15T10:05:00Z",
		CanceledAt:         "",
		CompletedAt:        "",
		RowsIngestedCount:  500,
		RowsInitialCount:   1000,
		RowsProcessedCount: 750,
		SortColumn:         "id",
	}

	model := &BackfillResourceModel{
		SinkConsumer: types.StringValue("my-consumer"),
	}

	mapBackfillResponseToModel(response, model)

	if model.ID.ValueString() != "bf-001" {
		t.Errorf("ID = %q, want bf-001", model.ID.ValueString())
	}
	if model.State.ValueString() != "active" {
		t.Errorf("State = %q, want active", model.State.ValueString())
	}
	if model.Table.ValueString() != "public.users" {
		t.Errorf("Table = %q, want public.users", model.Table.ValueString())
	}
	if model.SinkConsumer.ValueString() != "my-consumer" {
		t.Errorf("SinkConsumer = %q, want my-consumer (preserved from state)", model.SinkConsumer.ValueString())
	}

	if model.Status == nil {
		t.Fatal("Status should not be nil")
	}
	if model.Status.State != "active" {
		t.Errorf("Status.State = %q, want active", model.Status.State)
	}
	if model.Status.RowsIngestedCount != 500 {
		t.Errorf("RowsIngestedCount = %d, want 500", model.Status.RowsIngestedCount)
	}
	if model.Status.RowsInitialCount != 1000 {
		t.Errorf("RowsInitialCount = %d, want 1000", model.Status.RowsInitialCount)
	}
	if model.Status.RowsProcessedCount != 750 {
		t.Errorf("RowsProcessedCount = %d, want 750", model.Status.RowsProcessedCount)
	}
	if model.Status.SortColumn != "id" {
		t.Errorf("SortColumn = %q, want id", model.Status.SortColumn)
	}
}

func TestMapBackfillResponseToModel_NullSinkConsumerUsesAPI(t *testing.T) {
	response := &client.BackfillResponse{
		ID:           "bf-002",
		State:        "completed",
		Table:        "public.orders",
		SinkConsumer: "api-returned-name",
	}

	model := &BackfillResourceModel{
		SinkConsumer: types.StringNull(),
	}

	mapBackfillResponseToModel(response, model)

	if model.SinkConsumer.ValueString() != "api-returned-name" {
		t.Errorf("SinkConsumer = %q, want api-returned-name (from API when state is null)", model.SinkConsumer.ValueString())
	}
}

func TestMapBackfillResponseToModel_PreservesSinkConsumerFromState(t *testing.T) {
	response := &client.BackfillResponse{
		ID:           "bf-003",
		State:        "active",
		Table:        "public.users",
		SinkConsumer: "api-name-different",
	}

	model := &BackfillResourceModel{
		SinkConsumer: types.StringValue("my-consumer-id"),
	}

	mapBackfillResponseToModel(response, model)

	if model.SinkConsumer.ValueString() != "my-consumer-id" {
		t.Errorf("SinkConsumer = %q, want my-consumer-id (preserved from state)", model.SinkConsumer.ValueString())
	}
}

func TestMapBackfillResponseToModel_CompletedBackfill(t *testing.T) {
	response := &client.BackfillResponse{
		ID:                 "bf-004",
		State:              "completed",
		Table:              "public.events",
		SinkConsumer:       "event-processor",
		InsertedAt:         "2025-01-15T10:00:00Z",
		UpdatedAt:          "2025-01-15T12:00:00Z",
		CompletedAt:        "2025-01-15T12:00:00Z",
		RowsIngestedCount:  10000,
		RowsInitialCount:   10000,
		RowsProcessedCount: 10000,
		SortColumn:         "created_at",
	}

	model := &BackfillResourceModel{
		SinkConsumer: types.StringValue("event-processor"),
	}

	mapBackfillResponseToModel(response, model)

	if model.Status.CompletedAt != "2025-01-15T12:00:00Z" {
		t.Errorf("CompletedAt = %q, want 2025-01-15T12:00:00Z", model.Status.CompletedAt)
	}
	if model.Status.RowsProcessedCount != 10000 {
		t.Errorf("RowsProcessedCount = %d, want 10000", model.Status.RowsProcessedCount)
	}
}

func TestMapBackfillResponseToModel_CancelledBackfill(t *testing.T) {
	response := &client.BackfillResponse{
		ID:                 "bf-005",
		State:              "cancelled",
		Table:              "public.users",
		SinkConsumer:       "my-consumer",
		InsertedAt:         "2025-01-15T10:00:00Z",
		UpdatedAt:          "2025-01-15T10:30:00Z",
		CanceledAt:         "2025-01-15T10:30:00Z",
		CompletedAt:        "",
		RowsIngestedCount:  200,
		RowsInitialCount:   1000,
		RowsProcessedCount: 300,
	}

	model := &BackfillResourceModel{
		SinkConsumer: types.StringValue("my-consumer"),
	}

	mapBackfillResponseToModel(response, model)

	if model.State.ValueString() != "cancelled" {
		t.Errorf("State = %q, want cancelled", model.State.ValueString())
	}
	if model.Status.CanceledAt != "2025-01-15T10:30:00Z" {
		t.Errorf("CanceledAt = %q, want 2025-01-15T10:30:00Z", model.Status.CanceledAt)
	}
	if model.Status.CompletedAt != "" {
		t.Errorf("CompletedAt = %q, want empty", model.Status.CompletedAt)
	}
}
