package resources

import (
	"context"
	"testing"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	if consumerResource.client != mockClient {
		t.Error("Configure() did not set client correctly")
	}
}

func TestSinkConsumerResource_ConfigureWithInvalidType(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	configResp := &resource.ConfigureResponse{}
	consumerResource.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Error("Configure() with invalid type should error")
	}
}

func TestSinkConsumerResource_Metadata(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	resp := &resource.MetadataResponse{}
	consumerResource.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "sequin"}, resp)

	if resp.TypeName != "sequin_sink_consumer" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "sequin_sink_consumer")
	}
}

func TestSinkConsumerResource_Schema(t *testing.T) {
	ctx := context.Background()
	consumerResource := NewSinkConsumerResource().(*SinkConsumerResource)

	resp := &resource.SchemaResponse{}
	consumerResource.Schema(ctx, resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() error: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{
		"id", "name", "status", "database", "source", "tables", "actions",
		"destination", "filter", "transform", "enrichment", "routing",
		"message_grouping", "batch_size", "max_retry_count",
		"load_shedding_policy", "timestamp_format", "status_info",
	}
	for _, attr := range requiredAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("Schema() missing attribute: %s", attr)
		}
	}
}

// --- mapResponseToModel tests ---

// destAttrTypes is the attribute type map for destination objects
var destAttrTypes = map[string]attr.Type{
	"type":                  types.StringType,
	"hosts":                 types.StringType,
	"topic":                 types.StringType,
	"tls":                   types.BoolType,
	"username":              types.StringType,
	"password":              types.StringType,
	"sasl_mechanism":        types.StringType,
	"aws_region":            types.StringType,
	"aws_access_key_id":     types.StringType,
	"aws_secret_access_key": types.StringType,
	"queue_url":             types.StringType,
	"region":                types.StringType,
	"access_key_id":         types.StringType,
	"secret_access_key":     types.StringType,
	"is_fifo":               types.BoolType,
	"stream_arn":            types.StringType,
	"http_endpoint":         types.StringType,
	"http_endpoint_path":    types.StringType,
	"batch":                 types.BoolType,
}

func newNullDestModel() types.Object {
	return types.ObjectNull(destAttrTypes)
}

func TestMapResponseToModel_KafkaDestination(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	tls := true
	response := &client.SinkConsumerResponse{
		ID:       "sink-001",
		Name:     "kafka-sink",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert", "update"},
		Destination: client.SinkConsumerDestination{
			Type:  "kafka",
			Hosts: "broker1:9092,broker2:9092",
			Topic: "user-events",
			TLS:   &tls,
		},
		Filter:             "none",
		Transform:          "none",
		Enrichment:         "none",
		Routing:            "none",
		MessageGrouping:    true,
		BatchSize:          100,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	if model.ID.ValueString() != "sink-001" {
		t.Errorf("ID = %q, want sink-001", model.ID.ValueString())
	}
	if model.Name.ValueString() != "kafka-sink" {
		t.Errorf("Name = %q, want kafka-sink", model.Name.ValueString())
	}

	// Verify destination attributes
	destAttrs := model.Destination.Attributes()
	if destType, ok := destAttrs["type"].(types.String); !ok || destType.ValueString() != "kafka" {
		t.Errorf("destination type = %v, want kafka", destAttrs["type"])
	}
	if hosts, ok := destAttrs["hosts"].(types.String); !ok || hosts.ValueString() != "broker1:9092,broker2:9092" {
		t.Errorf("destination hosts = %v, want broker1:9092,broker2:9092", destAttrs["hosts"])
	}
	if tlsVal, ok := destAttrs["tls"].(types.Bool); !ok || tlsVal.ValueBool() != true {
		t.Errorf("destination tls = %v, want true", destAttrs["tls"])
	}
	// SQS fields should be null for kafka
	if queueURL, ok := destAttrs["queue_url"].(types.String); !ok || !queueURL.IsNull() {
		t.Errorf("destination queue_url should be null for kafka, got %v", destAttrs["queue_url"])
	}
}

func TestMapResponseToModel_NoneToNull(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	response := &client.SinkConsumerResponse{
		ID:       "sink-002",
		Name:     "test",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:         "webhook",
			HTTPEndpoint: "https://example.com",
		},
		Filter:             "none",
		Transform:          "none",
		Enrichment:         "none",
		Routing:            "none",
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	// "none" values should be mapped to null
	if !model.Filter.IsNull() {
		t.Errorf("Filter should be null when API returns 'none', got %q", model.Filter.ValueString())
	}
	if !model.Transform.IsNull() {
		t.Errorf("Transform should be null when API returns 'none', got %q", model.Transform.ValueString())
	}
	if !model.Enrichment.IsNull() {
		t.Errorf("Enrichment should be null when API returns 'none', got %q", model.Enrichment.ValueString())
	}
	if !model.Routing.IsNull() {
		t.Errorf("Routing should be null when API returns 'none', got %q", model.Routing.ValueString())
	}
}

func TestMapResponseToModel_ActualFilterValues(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	response := &client.SinkConsumerResponse{
		ID:       "sink-003",
		Name:     "filtered-sink",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.orders"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:         "webhook",
			HTTPEndpoint: "https://example.com",
		},
		Filter:             "record.status == 'active'",
		Transform:          "record.id",
		Enrichment:         "record",
		Routing:            "record.region",
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	if model.Filter.ValueString() != "record.status == 'active'" {
		t.Errorf("Filter = %q, want %q", model.Filter.ValueString(), "record.status == 'active'")
	}
	if model.Transform.ValueString() != "record.id" {
		t.Errorf("Transform = %q, want %q", model.Transform.ValueString(), "record.id")
	}
	if model.Routing.ValueString() != "record.region" {
		t.Errorf("Routing = %q, want %q", model.Routing.ValueString(), "record.region")
	}
}

func TestMapResponseToModel_SensitiveFieldPreservation(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	response := &client.SinkConsumerResponse{
		ID:       "sink-004",
		Name:     "kafka-sink",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:  "kafka",
			Hosts: "broker:9092",
			Topic: "events",
			// API does NOT return password or AWS keys
		},
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	// Simulate existing state with sensitive values
	allNullAttrs := map[string]attr.Value{
		"type":                  types.StringValue("kafka"),
		"hosts":                 types.StringValue("broker:9092"),
		"topic":                 types.StringValue("events"),
		"tls":                   types.BoolNull(),
		"username":              types.StringNull(),
		"password":              types.StringValue("my-secret-password"),
		"sasl_mechanism":        types.StringNull(),
		"aws_region":            types.StringNull(),
		"aws_access_key_id":     types.StringValue("AKIAIOSFODNN7"),
		"aws_secret_access_key": types.StringValue("wJalrXUtnFEMI/K7MDENG"),
		"queue_url":             types.StringNull(),
		"region":                types.StringNull(),
		"access_key_id":         types.StringNull(),
		"secret_access_key":     types.StringNull(),
		"is_fifo":               types.BoolNull(),
		"stream_arn":            types.StringNull(),
		"http_endpoint":         types.StringNull(),
		"http_endpoint_path":    types.StringNull(),
		"batch":                 types.BoolNull(),
	}
	existingDest, _ := types.ObjectValue(destAttrTypes, allNullAttrs)

	model := &SinkConsumerResourceModel{
		Destination: existingDest,
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	// Sensitive fields should be preserved from state
	destAttrs := model.Destination.Attributes()
	if password, ok := destAttrs["password"].(types.String); !ok || password.ValueString() != "my-secret-password" {
		t.Errorf("password should be preserved from state, got %v", destAttrs["password"])
	}
	if awsKey, ok := destAttrs["aws_access_key_id"].(types.String); !ok || awsKey.ValueString() != "AKIAIOSFODNN7" {
		t.Errorf("aws_access_key_id should be preserved from state, got %v", destAttrs["aws_access_key_id"])
	}
	if awsSecret, ok := destAttrs["aws_secret_access_key"].(types.String); !ok || awsSecret.ValueString() != "wJalrXUtnFEMI/K7MDENG" {
		t.Errorf("aws_secret_access_key should be preserved from state, got %v", destAttrs["aws_secret_access_key"])
	}
}

func TestMapResponseToModel_SQSDestination(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	isFifo := true
	response := &client.SinkConsumerResponse{
		ID:       "sink-005",
		Name:     "sqs-sink",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:     "sqs",
			QueueURL: "https://sqs.us-east-1.amazonaws.com/123/my-queue.fifo",
			Region:   "us-east-1",
			IsFIFO:   &isFifo,
		},
		BatchSize:          10,
		LoadSheddingPolicy: "discard_on_full",
		TimestampFormat:    "unix_microsecond",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	destAttrs := model.Destination.Attributes()
	if destType, ok := destAttrs["type"].(types.String); !ok || destType.ValueString() != "sqs" {
		t.Errorf("destination type = %v, want sqs", destAttrs["type"])
	}
	if queueURL, ok := destAttrs["queue_url"].(types.String); !ok || queueURL.ValueString() != "https://sqs.us-east-1.amazonaws.com/123/my-queue.fifo" {
		t.Errorf("queue_url = %v", destAttrs["queue_url"])
	}
	if isFifoVal, ok := destAttrs["is_fifo"].(types.Bool); !ok || isFifoVal.ValueBool() != true {
		t.Errorf("is_fifo = %v, want true", destAttrs["is_fifo"])
	}
	// Kafka fields should be null
	if hosts, ok := destAttrs["hosts"].(types.String); !ok || !hosts.IsNull() {
		t.Errorf("hosts should be null for SQS, got %v", destAttrs["hosts"])
	}
}

func TestMapResponseToModel_EmptySourceIsNull(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	response := &client.SinkConsumerResponse{
		ID:       "sink-006",
		Name:     "test",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:         "webhook",
			HTTPEndpoint: "https://example.com",
		},
		Source:             &client.SinkConsumerSource{}, // empty source
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	// Empty source should be null to avoid drift
	if !model.Source.IsNull() {
		t.Error("empty source should be mapped to null")
	}
}

func TestMapResponseToModel_SourceWithFilters(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	response := &client.SinkConsumerResponse{
		ID:       "sink-007",
		Name:     "test",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:         "webhook",
			HTTPEndpoint: "https://example.com",
		},
		Source: &client.SinkConsumerSource{
			IncludeSchemas: []string{"public", "app"},
		},
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("mapResponseToModel() errors: %v", diags.Errors())
	}

	if model.Source.IsNull() {
		t.Fatal("source with filters should not be null")
	}
}

func TestMapResponseToModel_MaxRetryCount(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	maxRetry := 5
	response := &client.SinkConsumerResponse{
		ID:       "sink-008",
		Name:     "test",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.users"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:         "webhook",
			HTTPEndpoint: "https://example.com",
		},
		MaxRetryCount:      &maxRetry,
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	model := &SinkConsumerResourceModel{
		Destination: newNullDestModel(),
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	if model.MaxRetryCount.ValueInt64() != 5 {
		t.Errorf("MaxRetryCount = %d, want 5", model.MaxRetryCount.ValueInt64())
	}

	// Test nil max_retry_count
	diags = diag.Diagnostics{}
	response.MaxRetryCount = nil
	model2 := &SinkConsumerResourceModel{Destination: newNullDestModel()}
	r.mapResponseToModel(ctx, response, model2, &diags)

	if !model2.MaxRetryCount.IsNull() {
		t.Error("nil MaxRetryCount should be mapped to null")
	}
}

func TestMapResponseToModel_TopicPreservationWithRouting(t *testing.T) {
	ctx := context.Background()
	r := &SinkConsumerResource{}
	diags := diag.Diagnostics{}

	// API returns empty topic when routing overrides it
	response := &client.SinkConsumerResponse{
		ID:       "sink-009",
		Name:     "routed-sink",
		Status:   "active",
		Database: "db-001",
		Tables:   []client.SinkConsumerTable{{Name: "public.events"}},
		Actions:  []string{"insert"},
		Destination: client.SinkConsumerDestination{
			Type:  "kafka",
			Hosts: "broker:9092",
			Topic: "", // empty because routing overrides
		},
		Routing:            "record.topic_name",
		BatchSize:          1,
		LoadSheddingPolicy: "pause_on_full",
		TimestampFormat:    "iso8601",
	}

	// State has the original topic
	stateAttrs := map[string]attr.Value{
		"type":                  types.StringValue("kafka"),
		"hosts":                 types.StringValue("broker:9092"),
		"topic":                 types.StringValue("default-topic"),
		"tls":                   types.BoolNull(),
		"username":              types.StringNull(),
		"password":              types.StringNull(),
		"sasl_mechanism":        types.StringNull(),
		"aws_region":            types.StringNull(),
		"aws_access_key_id":     types.StringNull(),
		"aws_secret_access_key": types.StringNull(),
		"queue_url":             types.StringNull(),
		"region":                types.StringNull(),
		"access_key_id":         types.StringNull(),
		"secret_access_key":     types.StringNull(),
		"is_fifo":               types.BoolNull(),
		"stream_arn":            types.StringNull(),
		"http_endpoint":         types.StringNull(),
		"http_endpoint_path":    types.StringNull(),
		"batch":                 types.BoolNull(),
	}
	existingDest, _ := types.ObjectValue(destAttrTypes, stateAttrs)

	model := &SinkConsumerResourceModel{
		Destination: existingDest,
	}

	r.mapResponseToModel(ctx, response, model, &diags)

	if diags.HasError() {
		t.Fatalf("errors: %v", diags.Errors())
	}

	// Topic should be preserved from state when API returns empty
	destAttrs := model.Destination.Attributes()
	if topic, ok := destAttrs["topic"].(types.String); !ok || topic.ValueString() != "default-topic" {
		t.Errorf("topic should be preserved from state when empty, got %v", destAttrs["topic"])
	}
}
