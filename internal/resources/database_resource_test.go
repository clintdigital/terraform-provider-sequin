package resources

import (
	"context"
	"testing"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDatabaseResource_Configure(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	// nil provider data
	configResp := &resource.ConfigureResponse{}
	dbResource.Configure(ctx, resource.ConfigureRequest{ProviderData: nil}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() with nil should not error, got: %v", configResp.Diagnostics.Errors())
	}

	// correct client type
	mockClient := &client.Client{}
	configResp = &resource.ConfigureResponse{}
	dbResource.Configure(ctx, resource.ConfigureRequest{ProviderData: mockClient}, configResp)
	if configResp.Diagnostics.HasError() {
		t.Errorf("Configure() error: %v", configResp.Diagnostics.Errors())
	}
	if dbResource.client != mockClient {
		t.Error("Configure() did not set client")
	}
}

func TestDatabaseResource_ConfigureWithInvalidType(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	configResp := &resource.ConfigureResponse{}
	dbResource.Configure(ctx, resource.ConfigureRequest{ProviderData: 42}, configResp)
	if !configResp.Diagnostics.HasError() {
		t.Error("Configure() with invalid type should error")
	}
}

func TestDatabaseResource_Metadata(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	resp := &resource.MetadataResponse{}
	dbResource.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sequin"}, resp)

	if resp.TypeName != "sequin_database" {
		t.Errorf("TypeName = %q, want sequin_database", resp.TypeName)
	}
}

func TestDatabaseResource_Schema(t *testing.T) {
	ctx := context.Background()
	dbResource := NewDatabaseResource().(*DatabaseResource)

	resp := &resource.SchemaResponse{}
	dbResource.Schema(ctx, resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() error: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{
		"id", "name", "url", "hostname", "port", "database", "username", "password",
		"ssl", "ipv6", "replication_slots", "primary",
		"use_local_tunnel", "pool_size", "queue_interval", "queue_target",
	}
	for _, attr := range requiredAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Schema() missing attribute: %s", attr)
		}
	}
}

// --- mapResponseToModel tests ---

func TestDatabaseMapResponseToModel_BasicFields(t *testing.T) {
	ctx := context.Background()
	r := &DatabaseResource{}
	diags := diag.Diagnostics{}

	response := &client.DatabaseResponse{
		ID:             "db-001",
		Name:           "production",
		Hostname:       "db.example.com",
		Port:           5432,
		Database:       "myapp",
		Username:       "admin",
		Password:       "***obfuscated***",
		SSL:            true,
		IPv6:           false,
		UseLocalTunnel: false,
		PoolSize:       10,
		QueueInterval:  1000,
		QueueTarget:    500,
		ReplicationSlots: []client.ReplicationSlot{
			{
				ID:              "slot-001",
				PublicationName: "sequin_pub",
				SlotName:        "sequin_slot",
				Status:          "active",
			},
		},
	}

	model := &DatabaseResourceModel{
		Password: types.StringValue("real-password"),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	if model.ID.ValueString() != "db-001" {
		t.Errorf("ID = %q, want db-001", model.ID.ValueString())
	}
	if model.Name.ValueString() != "production" {
		t.Errorf("Name = %q, want production", model.Name.ValueString())
	}
	if model.Hostname.ValueString() != "db.example.com" {
		t.Errorf("Hostname = %q, want db.example.com", model.Hostname.ValueString())
	}
	if model.Port.ValueInt64() != 5432 {
		t.Errorf("Port = %d, want 5432", model.Port.ValueInt64())
	}
	if model.SSL.ValueBool() != true {
		t.Error("SSL should be true")
	}

	// Computed fields
	if model.PoolSize.ValueInt64() != 10 {
		t.Errorf("PoolSize = %d, want 10", model.PoolSize.ValueInt64())
	}
	if model.QueueInterval.ValueInt64() != 1000 {
		t.Errorf("QueueInterval = %d, want 1000", model.QueueInterval.ValueInt64())
	}
	if model.UseLocalTunnel.ValueBool() != false {
		t.Error("UseLocalTunnel should be false")
	}
}

func TestDatabaseMapResponseToModel_ReplicationSlots(t *testing.T) {
	ctx := context.Background()
	r := &DatabaseResource{}
	diags := diag.Diagnostics{}

	response := &client.DatabaseResponse{
		ID:       "db-002",
		Name:     "test",
		Hostname: "localhost",
		Port:     5432,
		ReplicationSlots: []client.ReplicationSlot{
			{
				ID:              "slot-001",
				PublicationName: "pub1",
				SlotName:        "slot1",
				Status:          "active",
			},
			{
				ID:              "slot-002",
				PublicationName: "pub2",
				SlotName:        "slot2",
				Status:          "", // empty status
			},
		},
	}

	model := &DatabaseResourceModel{}
	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	if model.ReplicationSlots.IsNull() {
		t.Fatal("ReplicationSlots should not be null")
	}
	if len(model.ReplicationSlots.Elements()) != 2 {
		t.Fatalf("ReplicationSlots length = %d, want 2", len(model.ReplicationSlots.Elements()))
	}
}

func TestDatabaseMapResponseToModel_PrimaryPresent(t *testing.T) {
	ctx := context.Background()
	r := &DatabaseResource{}
	diags := diag.Diagnostics{}

	port := 5433
	ssl := true
	response := &client.DatabaseResponse{
		ID:               "db-003",
		Name:             "replica",
		Hostname:         "replica.example.com",
		Port:             5432,
		ReplicationSlots: []client.ReplicationSlot{},
		Primary: &client.PrimaryDatabase{
			Hostname: "primary.example.com",
			Database: "myapp",
			Username: "replicator",
			Password: "***",
			Port:     &port,
			SSL:      &ssl,
		},
	}

	model := &DatabaseResourceModel{}
	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	if model.Primary.IsNull() {
		t.Fatal("Primary should not be null when present in response")
	}

	primaryAttrs := model.Primary.Attributes()
	if hostname, ok := primaryAttrs["hostname"].(types.String); !ok || hostname.ValueString() != "primary.example.com" {
		t.Errorf("primary hostname = %v, want primary.example.com", primaryAttrs["hostname"])
	}
	if portVal, ok := primaryAttrs["port"].(types.Int64); !ok || portVal.ValueInt64() != 5433 {
		t.Errorf("primary port = %v, want 5433", primaryAttrs["port"])
	}
	if sslVal, ok := primaryAttrs["ssl"].(types.Bool); !ok || sslVal.ValueBool() != true {
		t.Errorf("primary ssl = %v, want true", primaryAttrs["ssl"])
	}
}

func TestDatabaseMapResponseToModel_PrimaryAbsent(t *testing.T) {
	ctx := context.Background()
	r := &DatabaseResource{}
	diags := diag.Diagnostics{}

	response := &client.DatabaseResponse{
		ID:               "db-004",
		Name:             "standalone",
		Hostname:         "db.example.com",
		Port:             5432,
		ReplicationSlots: []client.ReplicationSlot{},
		Primary:          nil,
	}

	model := &DatabaseResourceModel{}
	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	if !model.Primary.IsNull() {
		t.Error("Primary should be null when not present in response")
	}
}

func TestDatabaseMapResponseToModel_PrimaryNilPort(t *testing.T) {
	ctx := context.Background()
	r := &DatabaseResource{}
	diags := diag.Diagnostics{}

	response := &client.DatabaseResponse{
		ID:               "db-005",
		Name:             "replica",
		Hostname:         "replica.example.com",
		Port:             5432,
		ReplicationSlots: []client.ReplicationSlot{},
		Primary: &client.PrimaryDatabase{
			Hostname: "primary.example.com",
			Database: "myapp",
			Username: "replicator",
			Password: "***",
			Port:     nil, // nil port
			SSL:      nil, // nil ssl
		},
	}

	model := &DatabaseResourceModel{}
	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	primaryAttrs := model.Primary.Attributes()
	if portVal, ok := primaryAttrs["port"].(types.Int64); !ok || !portVal.IsNull() {
		t.Errorf("primary port should be null when nil, got %v", primaryAttrs["port"])
	}
	if sslVal, ok := primaryAttrs["ssl"].(types.Bool); !ok || !sslVal.IsNull() {
		t.Errorf("primary ssl should be null when nil, got %v", primaryAttrs["ssl"])
	}
}
