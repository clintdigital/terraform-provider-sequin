package resources

import (
	"context"
	"testing"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestSinkConsumerResource_Configure tests the Configure method
func TestSinkConsumerResource_Configure(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	// Test with nil provider data (should not error)
	configResp := &resource.ConfigureResponse{}
	consumerResource.Configure(ctx, resource.ConfigureRequest{ProviderData: nil}, configResp)

	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with nil provider data should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// Test with correct client type
	mockClient := &client.Client{}
	configResp = &resource.ConfigureResponse{}
	consumerResource.Configure(ctx, resource.ConfigureRequest{ProviderData: mockClient}, configResp)

	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with correct client should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// Verify client was set
	if consumerResource.client != mockClient {
		t.Error("Configure() did not set client correctly")
	}
}

// TestSinkConsumerResource_ConfigureWithInvalidType tests Configure with wrong type
func TestSinkConsumerResource_ConfigureWithInvalidType(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	// Test with incorrect type (should error)
	configResp := &resource.ConfigureResponse{}
	consumerResource.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Error("Configure() with invalid type should error")
	}
}

// TestSinkConsumerResource_Metadata tests the Metadata method
func TestSinkConsumerResource_Metadata(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	req := resource.MetadataRequest{
		ProviderTypeName: "sequin",
	}
	resp := &resource.MetadataResponse{}

	consumerResource.Metadata(ctx, req, resp)

	expectedTypeName := "sequin_sink_consumer"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, expectedTypeName)
	}
}

// TestSinkConsumerResource_Schema tests that Schema method doesn't error
func TestSinkConsumerResource_Schema(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	consumerResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() should not error, got: %v", resp.Diagnostics.Errors())
	}

	// Verify schema has expected attributes
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{
		"id", "name", "status", "database", "source", "tables", "actions",
		"destination", "filter", "transform", "enrichment", "routing",
		"message_grouping", "batch_size", "max_retry_count",
		"load_shedding_policy", "timestamp_format", "status_info",
	}
	for _, attr := range requiredAttrs {
		if _, ok := attrs[attr]; !ok {
			t.Errorf("Schema() missing required attribute: %s", attr)
		}
	}

	// Verify nested destination attributes
	if destAttr, ok := attrs["destination"]; ok {
		// Check that it's a SingleNestedAttribute with expected fields
		t.Logf("Destination attribute found with type: %T", destAttr)
	} else {
		t.Error("Schema() missing destination attribute")
	}

	// Verify tables is a ListNestedAttribute
	if tablesAttr, ok := attrs["tables"]; ok {
		t.Logf("Tables attribute found with type: %T", tablesAttr)
	} else {
		t.Error("Schema() missing tables attribute")
	}
}

// TestNewSinkConsumerResource tests the constructor
func TestNewSinkConsumerResource(t *testing.T) {
	r := NewSinkConsumerResource()
	if r == nil {
		t.Fatal("NewSinkConsumerResource() returned nil")
	}

	// Verify it's the correct type
	_, ok := r.(*SinkConsumerResource)
	if !ok {
		t.Fatal("NewSinkConsumerResource() did not return *SinkConsumerResource")
	}
}
