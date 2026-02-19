package resources

import (
	"context"
	"testing"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestDatabaseResource_Configure tests the Configure method
func TestDatabaseResource_Configure(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	// Test with nil provider data (should not error)
	configResp := &resource.ConfigureResponse{}
	dbResource.Configure(ctx, resource.ConfigureRequest{ProviderData: nil}, configResp)

	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with nil provider data should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// Test with correct client type
	mockClient := &client.Client{}
	configResp = &resource.ConfigureResponse{}
	dbResource.Configure(ctx, resource.ConfigureRequest{ProviderData: mockClient}, configResp)

	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with correct client should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// Verify client was set
	if dbResource.client != mockClient {
		t.Error("Configure() did not set client correctly")
	}
}

// TestDatabaseResource_ConfigureWithInvalidType tests Configure with wrong type
func TestDatabaseResource_ConfigureWithInvalidType(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	// Test with incorrect type (should error)
	configResp := &resource.ConfigureResponse{}
	dbResource.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Error("Configure() with invalid type should error")
	}
}

// TestDatabaseResource_Metadata tests the Metadata method
func TestDatabaseResource_Metadata(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	req := resource.MetadataRequest{
		ProviderTypeName: "sequin",
	}
	resp := &resource.MetadataResponse{}

	dbResource.Metadata(ctx, req, resp)

	expectedTypeName := "sequin_database"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, expectedTypeName)
	}
}

// TestDatabaseResource_Schema tests that Schema method doesn't error
func TestDatabaseResource_Schema(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	dbResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() should not error, got: %v", resp.Diagnostics.Errors())
	}

	// Verify schema has expected attributes
	attrs := resp.Schema.Attributes
	requiredAttrs := []string{
		"id", "name", "url", "hostname", "port", "database", "username", "password",
		"ssl", "ipv6", "replication_slots", "primary",
		"use_local_tunnel", "pool_size", "queue_interval", "queue_target",
	}
	for _, attr := range requiredAttrs {
		if _, ok := attrs[attr]; !ok {
			t.Errorf("Schema() missing required attribute: %s", attr)
		}
	}
}

// TestNewDatabaseResource tests the constructor
func TestNewDatabaseResource(t *testing.T) {
	r := NewDatabaseResource()
	if r == nil {
		t.Fatal("NewDatabaseResource() returned nil")
	}

	// Verify it's the correct type
	_, ok := r.(*DatabaseResource)
	if !ok {
		t.Fatal("NewDatabaseResource() did not return *DatabaseResource")
	}
}
