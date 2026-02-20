package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	c := New("https://api.example.com", "test-key", "0.1.0")

	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.example.com")
	}
	if c.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "test-key")
	}
	if c.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", c.Version, "0.1.0")
	}
	if c.HTTPClient == nil {
		t.Fatal("HTTPClient should not be nil")
	}
}

func TestDoRequest_SetsAuthHeaders(t *testing.T) {
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := New(server.URL, "my-secret-key", "1.2.3")
	_, err := c.doRequest(context.Background(), http.MethodGet, "/api/test", nil)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	if got := capturedReq.Header.Get("Authorization"); got != "Bearer my-secret-key" {
		t.Errorf("Authorization header = %q, want %q", got, "Bearer my-secret-key")
	}
	if got := capturedReq.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := capturedReq.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want %q", got, "application/json")
	}
	if got := capturedReq.Header.Get("User-Agent"); got != "terraform-provider-sequin/1.2.3" {
		t.Errorf("User-Agent = %q, want %q", got, "terraform-provider-sequin/1.2.3")
	}
}

func TestDoRequest_MarshalsBody(t *testing.T) {
	var capturedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	body := map[string]string{"name": "test-db"}
	_, err := c.doRequest(context.Background(), http.MethodPost, "/api/databases", body)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	if capturedBody["name"] != "test-db" {
		t.Errorf("body name = %q, want %q", capturedBody["name"], "test-db")
	}
}

func TestHandleResponse_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error": "validation failed"}`))
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	resp, err := c.doRequest(context.Background(), http.MethodPost, "/api/test", nil)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	var target map[string]string
	err = c.handleResponse(context.Background(), resp, &target)
	if err == nil {
		t.Fatal("handleResponse() should return error for 422")
	}
	if got := err.Error(); got != `API error (status 422): {"error": "validation failed"}` {
		t.Errorf("error = %q", got)
	}
}

func TestHandleResponse_UnmarshalSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"db-123","name":"my-db"}`))
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	resp, err := c.doRequest(context.Background(), http.MethodGet, "/api/test", nil)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	var target DatabaseResponse
	err = c.handleResponse(context.Background(), resp, &target)
	if err != nil {
		t.Fatalf("handleResponse() error: %v", err)
	}
	if target.ID != "db-123" {
		t.Errorf("ID = %q, want %q", target.ID, "db-123")
	}
	if target.Name != "my-db" {
		t.Errorf("Name = %q, want %q", target.Name, "my-db")
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"not found message", fmt.Errorf("database not found: abc"), true},
		{"404 message", fmt.Errorf("API error (status 404): not found"), true},
		{"unrelated error", fmt.Errorf("connection refused"), false},
		{"sink not found", fmt.Errorf("sink consumer not found: xyz"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Database CRUD integration tests with httptest ---

func TestCreateDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/postgres_databases" {
			t.Errorf("path = %s, want /api/postgres_databases", r.URL.Path)
		}

		var req DatabaseRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "test-db" {
			t.Errorf("request name = %q, want %q", req.Name, "test-db")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DatabaseResponse{
			ID:       "db-001",
			Name:     "test-db",
			Hostname: "localhost",
			Port:     5432,
		})
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	resp, err := c.CreateDatabase(context.Background(), &DatabaseRequest{Name: "test-db"})
	if err != nil {
		t.Fatalf("CreateDatabase() error: %v", err)
	}
	if resp.ID != "db-001" {
		t.Errorf("ID = %q, want %q", resp.ID, "db-001")
	}
}

func TestGetDatabase_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	_, err := c.GetDatabase(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("GetDatabase() should return error for 404")
	}
	if !IsNotFoundError(err) {
		t.Errorf("error should be detected as not found: %v", err)
	}
}

func TestDeleteDatabase_AlreadyDeleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	err := c.DeleteDatabase(context.Background(), "already-gone")
	if err != nil {
		t.Errorf("DeleteDatabase() should not error for already-deleted resource, got: %v", err)
	}
}

// --- SinkConsumer CRUD tests ---

func TestCreateSinkConsumer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/sinks" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}

		var req SinkConsumerRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Destination.Type != "kafka" {
			t.Errorf("destination type = %q, want kafka", req.Destination.Type)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SinkConsumerResponse{
			ID:   "sink-001",
			Name: req.Name,
			Destination: SinkConsumerDestination{
				Type:  "kafka",
				Hosts: "broker:9092",
				Topic: "events",
			},
		})
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	resp, err := c.CreateSinkConsumer(context.Background(), &SinkConsumerRequest{
		Name:     "my-sink",
		Database: "db-001",
		Destination: SinkConsumerDestination{
			Type:  "kafka",
			Hosts: "broker:9092",
			Topic: "events",
		},
	})
	if err != nil {
		t.Fatalf("CreateSinkConsumer() error: %v", err)
	}
	if resp.Destination.Type != "kafka" {
		t.Errorf("destination type = %q, want kafka", resp.Destination.Type)
	}
}

func TestGetSinkConsumer_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	_, err := c.GetSinkConsumer(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("GetSinkConsumer() should return error for 404")
	}
}

func TestDeleteSinkConsumer_AlreadyDeleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	err := c.DeleteSinkConsumer(context.Background(), "already-gone")
	if err != nil {
		t.Errorf("DeleteSinkConsumer() should not error for already-deleted resource, got: %v", err)
	}
}

// --- Backfill CRUD tests ---

func TestCreateBackfill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sinks/my-sink/backfills" {
			t.Errorf("path = %s, want /api/sinks/my-sink/backfills", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BackfillResponse{
			ID:    "bf-001",
			State: "active",
			Table: "public.users",
		})
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	resp, err := c.CreateBackfill(context.Background(), "my-sink", &BackfillCreateRequest{
		Table: "public.users",
	})
	if err != nil {
		t.Fatalf("CreateBackfill() error: %v", err)
	}
	if resp.State != "active" {
		t.Errorf("state = %q, want active", resp.State)
	}
}

func TestListBackfills(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BackfillListResponse{
			Data: []BackfillResponse{
				{ID: "bf-001", State: "active"},
				{ID: "bf-002", State: "completed"},
			},
		})
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	backfills, err := c.ListBackfills(context.Background(), "my-sink")
	if err != nil {
		t.Fatalf("ListBackfills() error: %v", err)
	}
	if len(backfills) != 2 {
		t.Fatalf("got %d backfills, want 2", len(backfills))
	}
	if backfills[0].ID != "bf-001" {
		t.Errorf("first backfill ID = %q, want bf-001", backfills[0].ID)
	}
}

func TestUpdateBackfill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}

		var req BackfillUpdateRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.State != "cancelled" {
			t.Errorf("state = %q, want cancelled", req.State)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BackfillResponse{
			ID:    "bf-001",
			State: "cancelled",
		})
	}))
	defer server.Close()

	c := New(server.URL, "key", "1.0.0")
	resp, err := c.UpdateBackfill(context.Background(), "my-sink", "bf-001", &BackfillUpdateRequest{
		State: "cancelled",
	})
	if err != nil {
		t.Fatalf("UpdateBackfill() error: %v", err)
	}
	if resp.State != "cancelled" {
		t.Errorf("state = %q, want cancelled", resp.State)
	}
}

// --- JSON serialization tests ---

func TestDatabaseRequest_JSONOmitsEmpty(t *testing.T) {
	req := DatabaseRequest{
		Name:     "test",
		Hostname: "localhost",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	// URL should be omitted when empty
	if _, ok := m["url"]; ok {
		t.Error("empty url should be omitted from JSON")
	}
	// Port should be omitted when nil
	if _, ok := m["port"]; ok {
		t.Error("nil port should be omitted from JSON")
	}
}

func TestSinkConsumerDestination_PointerBooleans(t *testing.T) {
	tls := true
	dest := SinkConsumerDestination{
		Type:  "kafka",
		Hosts: "broker:9092",
		TLS:   &tls,
	}

	data, err := json.Marshal(dest)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["tls"] != true {
		t.Errorf("tls = %v, want true", m["tls"])
	}

	// SQS fields should be omitted
	if _, ok := m["queue_url"]; ok {
		t.Error("queue_url should be omitted for kafka destination")
	}
}

func TestSinkConsumerDestination_NilBooleans(t *testing.T) {
	dest := SinkConsumerDestination{
		Type:     "webhook",
		HTTPEndpoint: "https://example.com",
	}

	data, err := json.Marshal(dest)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	// Nil pointer bools should be omitted
	if _, ok := m["tls"]; ok {
		t.Error("nil tls should be omitted")
	}
	if _, ok := m["is_fifo"]; ok {
		t.Error("nil is_fifo should be omitted")
	}
	if _, ok := m["batch"]; ok {
		t.Error("nil batch should be omitted")
	}
}
