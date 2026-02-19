package resources

import (
	"context"
	"testing"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestBackfillResource_Configure tests the Configure method
func TestBackfillResource_Configure(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	// Test with nil provider data (should not error)
	configResp := &resource.ConfigureResponse{}
	backfillResource.Configure(ctx, resource.ConfigureRequest{ProviderData: nil}, configResp)

	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with nil provider data should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// Test with correct client type
	mockClient := &client.Client{}
	configResp = &resource.ConfigureResponse{}
	backfillResource.Configure(ctx, resource.ConfigureRequest{ProviderData: mockClient}, configResp)

	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with correct client should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// Verify client was set
	if backfillResource.client != mockClient {
		t.Error("Configure() did not set client correctly")
	}
}

// TestBackfillResource_ConfigureWithInvalidType tests Configure with wrong type
func TestBackfillResource_ConfigureWithInvalidType(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	// Test with incorrect type (should error)
	configResp := &resource.ConfigureResponse{}
	backfillResource.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Error("Configure() with invalid type should error")
	}
}

// TestBackfillResource_Metadata tests the Metadata method
func TestBackfillResource_Metadata(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	req := resource.MetadataRequest{
		ProviderTypeName: "sequin",
	}
	resp := &resource.MetadataResponse{}

	backfillResource.Metadata(ctx, req, resp)

	expectedTypeName := "sequin_backfill"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, expectedTypeName)
	}
}

// TestBackfillResource_Schema tests that Schema method doesn't error
func TestBackfillResource_Schema(t *testing.T) {
	ctx := context.Background()
	backfillResource := NewBackfillResource().(*BackfillResource)

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	backfillResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() should not error, got: %v", resp.Diagnostics.Errors())
	}

	// Verify schema has expected attributes
	attrs := resp.Schema.Attributes
	expectedAttrs := []string{"id", "sink_consumer", "table", "state", "status"}
	for _, attr := range expectedAttrs {
		if _, ok := attrs[attr]; !ok {
			t.Errorf("Schema() missing expected attribute: %s", attr)
		}
	}

	// Verify sink_consumer and table have RequiresReplace modifiers
	immutableFields := []string{"sink_consumer", "table"}
	for _, field := range immutableFields {
		if attr, ok := attrs[field]; ok {
			if stringAttr, ok := attr.(schema.StringAttribute); ok {
				if len(stringAttr.PlanModifiers) == 0 {
					t.Errorf("Schema() field %s should have plan modifiers", field)
				}
			}
		}
	}

	// Verify status is a nested attribute with expected fields
	statusAttr, ok := attrs["status"]
	if !ok {
		t.Fatal("Schema() missing status attribute")
	}
	nestedAttr, ok := statusAttr.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("Schema() status should be SingleNestedAttribute")
	}
	statusFields := []string{"state", "inserted_at", "updated_at", "canceled_at", "completed_at", "rows_ingested_count", "rows_initial_count", "rows_processed_count", "sort_column"}
	for _, field := range statusFields {
		if _, ok := nestedAttr.Attributes[field]; !ok {
			t.Errorf("Schema() status missing field: %s", field)
		}
	}
}

// TestNewBackfillResource tests the constructor
func TestNewBackfillResource(t *testing.T) {
	r := NewBackfillResource()
	if r == nil {
		t.Fatal("NewBackfillResource() returned nil")
	}

	// Verify it's the correct type
	_, ok := r.(*BackfillResource)
	if !ok {
		t.Fatal("NewBackfillResource() did not return *BackfillResource")
	}
}
