package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client handles communication with the Sequin API
type Client struct {
	BaseURL    string
	APIKey     string
	Version    string
	HTTPClient *http.Client
}

// New creates a new Sequin API client
func New(baseURL, apiKey, version string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Version: version,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication and logging
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("terraform-provider-sequin/%s", c.Version))

	tflog.Debug(ctx, "Making API request", map[string]any{
		"method": method,
		"url":    url,
	})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	tflog.Debug(ctx, "Received API response", map[string]any{
		"status_code": resp.StatusCode,
	})

	return resp, nil
}

// handleResponse processes the HTTP response and unmarshals into target
func (c *Client) handleResponse(ctx context.Context, resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		tflog.Error(ctx, "API error response", map[string]any{
			"status_code": resp.StatusCode,
		})
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Database API request/response structs

// ReplicationSlot represents a replication slot configuration
type ReplicationSlot struct {
	ID              string `json:"id,omitempty"`               // Computed (only in response/update)
	PublicationName string `json:"publication_name"`           // Required
	SlotName        string `json:"slot_name"`                  // Required
	Status          string `json:"status,omitempty"`           // Optional: active, disabled
}

// PrimaryDatabase represents the primary database configuration when connecting to a replica
type PrimaryDatabase struct {
	Hostname string `json:"hostname"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Port     *int   `json:"port,omitempty"`
	SSL      *bool  `json:"ssl,omitempty"`
}

// DatabaseRequest represents the request body for creating or updating a database
type DatabaseRequest struct {
	Name             string             `json:"name"`
	URL              string             `json:"url,omitempty"`               // Alternative to individual connection params
	Hostname         string             `json:"hostname,omitempty"`
	Port             *int               `json:"port,omitempty"`
	Database         string             `json:"database,omitempty"`
	Username         string             `json:"username,omitempty"`
	Password         string             `json:"password,omitempty"`
	SSL              *bool              `json:"ssl,omitempty"`
	IPv6             *bool              `json:"ipv6,omitempty"`
	ReplicationSlots []ReplicationSlot  `json:"replication_slots,omitempty"` // Required for create, optional for update
	Primary          *PrimaryDatabase   `json:"primary,omitempty"`           // For replica configuration
}

// DatabaseResponse represents a database resource from the API
type DatabaseResponse struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Hostname         string            `json:"hostname"`
	Port             int               `json:"port"`
	Database         string            `json:"database"`
	Username         string            `json:"username"`
	Password         string            `json:"password"`           // Obfuscated in response
	SSL              bool              `json:"ssl"`
	IPv6             bool              `json:"ipv6"`
	UseLocalTunnel   bool              `json:"use_local_tunnel"`   // Computed
	PoolSize         int               `json:"pool_size"`          // Computed
	QueueInterval    int               `json:"queue_interval"`     // Computed
	QueueTarget      int               `json:"queue_target"`       // Computed
	ReplicationSlots []ReplicationSlot `json:"replication_slots"`
	Primary          *PrimaryDatabase  `json:"primary,omitempty"`
}

// StatusResponse represents the status of a resource
type StatusResponse struct {
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	LastError string `json:"last_error,omitempty"`
}

// Database CRUD methods

// CreateDatabase creates a new database connection
func (c *Client) CreateDatabase(ctx context.Context, req *DatabaseRequest) (*DatabaseResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/postgres_databases", req)
	if err != nil {
		return nil, err
	}

	var result DatabaseResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	tflog.Info(ctx, "Created database", map[string]any{"id": result.ID, "name": result.Name})
	return &result, nil
}

// GetDatabase retrieves a database by ID
func (c *Client) GetDatabase(ctx context.Context, id string) (*DatabaseResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/postgres_databases/%s", id), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("database not found: %s", id)
	}

	var result DatabaseResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &result, nil
}

// UpdateDatabase updates an existing database
func (c *Client) UpdateDatabase(ctx context.Context, id string, req *DatabaseRequest) (*DatabaseResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/postgres_databases/%s", id), req)
	if err != nil {
		return nil, err
	}

	var result DatabaseResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to update database: %w", err)
	}

	tflog.Info(ctx, "Updated database", map[string]any{"id": result.ID})
	return &result, nil
}

// DeleteDatabase deletes a database by ID
func (c *Client) DeleteDatabase(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/postgres_databases/%s", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Database already deleted", map[string]any{"id": id})
		return nil
	}

	if err := c.handleResponse(ctx, resp, nil); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	tflog.Info(ctx, "Deleted database", map[string]any{"id": id})
	return nil
}

// SinkConsumer API request/response structs

// SinkConsumerTable represents a table configuration in a sink consumer
type SinkConsumerTable struct {
	Name              string   `json:"name"`
	GroupColumnNames  []string `json:"group_column_names,omitempty"`
}

// SinkConsumerSource represents the source configuration
type SinkConsumerSource struct {
	IncludeSchemas []string `json:"include_schemas,omitempty"`
	ExcludeSchemas []string `json:"exclude_schemas,omitempty"`
	IncludeTables  []string `json:"include_tables,omitempty"`
	ExcludeTables  []string `json:"exclude_tables,omitempty"`
}

// SinkConsumerDestination represents the destination configuration
type SinkConsumerDestination struct {
	Type string `json:"type"` // kafka, sqs, kinesis, webhook

	// Kafka fields
	Hosts              string `json:"hosts,omitempty"`
	Topic              string `json:"topic,omitempty"`
	TLS                *bool  `json:"tls,omitempty"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	SASLMechanism      string `json:"sasl_mechanism,omitempty"`
	AWSRegion          string `json:"aws_region,omitempty"`
	AWSAccessKeyID     string `json:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey string `json:"aws_secret_access_key,omitempty"`

	// SQS fields
	QueueURL        string `json:"queue_url,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	IsFIFO          *bool  `json:"is_fifo,omitempty"`

	// Kinesis fields
	StreamARN string `json:"stream_arn,omitempty"`

	// Webhook fields
	HTTPEndpoint     string `json:"http_endpoint,omitempty"`
	HTTPEndpointPath string `json:"http_endpoint_path,omitempty"`
	Batch            *bool  `json:"batch,omitempty"`
}

// SinkConsumerRequest represents the request body for creating or updating a sink consumer
type SinkConsumerRequest struct {
	Name               string                   `json:"name"`
	Status             string                   `json:"status,omitempty"`                // active, disabled, paused
	Database           string                   `json:"database"`
	Source             *SinkConsumerSource      `json:"source,omitempty"`
	Tables             []SinkConsumerTable      `json:"tables"`
	Actions            []string                 `json:"actions,omitempty"`                // insert, update, delete
	Destination        SinkConsumerDestination  `json:"destination"`
	Filter             string                   `json:"filter,omitempty"`
	Transform          string                   `json:"transform,omitempty"`
	Enrichment         string                   `json:"enrichment,omitempty"`
	Routing            string                   `json:"routing,omitempty"`
	MessageGrouping    *bool                    `json:"message_grouping,omitempty"`
	BatchSize          *int                     `json:"batch_size,omitempty"`
	MaxRetryCount      *int                     `json:"max_retry_count,omitempty"`
	LoadSheddingPolicy string                   `json:"load_shedding_policy,omitempty"` // pause_on_full, discard_on_full
	TimestampFormat    string                   `json:"timestamp_format,omitempty"`     // iso8601, unix_microsecond
}

// SinkConsumerResponse represents a sink consumer resource from the API
type SinkConsumerResponse struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Status             string                   `json:"status"`
	Database           string                   `json:"database"`
	Source             *SinkConsumerSource      `json:"source,omitempty"`
	Tables             []SinkConsumerTable      `json:"tables"`
	Actions            []string                 `json:"actions"`
	Destination        SinkConsumerDestination  `json:"destination"`
	Filter             string                   `json:"filter,omitempty"`
	Transform          string                   `json:"transform,omitempty"`
	Enrichment         string                   `json:"enrichment,omitempty"`
	Routing            string                   `json:"routing,omitempty"`
	MessageGrouping    bool                     `json:"message_grouping"`
	BatchSize          int                      `json:"batch_size"`
	MaxRetryCount      *int                     `json:"max_retry_count,omitempty"`
	LoadSheddingPolicy string                   `json:"load_shedding_policy"`
	TimestampFormat    string                   `json:"timestamp_format"`
	StatusInfo         StatusResponse           `json:"status_info"`
}

// SinkConsumer CRUD methods

// CreateSinkConsumer creates a new sink consumer
func (c *Client) CreateSinkConsumer(ctx context.Context, req *SinkConsumerRequest) (*SinkConsumerResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/sinks", req)
	if err != nil {
		return nil, err
	}

	var result SinkConsumerResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to create sink consumer: %w", err)
	}

	tflog.Info(ctx, "Created sink consumer", map[string]any{"id": result.ID, "name": result.Name})
	return &result, nil
}

// GetSinkConsumer retrieves a sink consumer by ID
func (c *Client) GetSinkConsumer(ctx context.Context, id string) (*SinkConsumerResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/sinks/%s", id), nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("sink consumer not found: %s", id)
	}

	var result SinkConsumerResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to get sink consumer: %w", err)
	}

	return &result, nil
}

// UpdateSinkConsumer updates an existing sink consumer
func (c *Client) UpdateSinkConsumer(ctx context.Context, id string, req *SinkConsumerRequest) (*SinkConsumerResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/sinks/%s", id), req)
	if err != nil {
		return nil, err
	}

	var result SinkConsumerResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to update sink consumer: %w", err)
	}

	tflog.Info(ctx, "Updated sink consumer", map[string]any{"id": result.ID})
	return &result, nil
}

// DeleteSinkConsumer deletes a sink consumer by ID
func (c *Client) DeleteSinkConsumer(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/sinks/%s", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Sink consumer already deleted", map[string]any{"id": id})
		return nil
	}

	if err := c.handleResponse(ctx, resp, nil); err != nil {
		return fmt.Errorf("failed to delete sink consumer: %w", err)
	}

	tflog.Info(ctx, "Deleted sink consumer", map[string]any{"id": id})
	return nil
}

// Backfill API request/response structs

// BackfillCreateRequest represents the request body for creating a backfill
type BackfillCreateRequest struct {
	Table string `json:"table,omitempty"` // schema.table format, optional if sink has single table
}

// BackfillUpdateRequest represents the request body for updating a backfill
type BackfillUpdateRequest struct {
	State string `json:"state"` // active, cancelled
}

// BackfillResponse represents a backfill resource from the API
type BackfillResponse struct {
	ID                 string `json:"id"`
	State              string `json:"state"`               // active, completed, cancelled
	Table              string `json:"table"`
	InsertedAt         string `json:"inserted_at"`
	SinkConsumer       string `json:"sink_consumer"`       // sink consumer name
	UpdatedAt          string `json:"updated_at"`
	CanceledAt         string `json:"canceled_at"`
	CompletedAt        string `json:"completed_at"`
	RowsIngestedCount  int    `json:"rows_ingested_count"`
	RowsInitialCount   int    `json:"rows_initial_count"`
	RowsProcessedCount int    `json:"rows_processed_count"`
	SortColumn         string `json:"sort_column"`
}

// BackfillDeleteResponse represents the response from deleting a backfill
type BackfillDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// BackfillListResponse represents the response from listing backfills
type BackfillListResponse struct {
	Data []BackfillResponse `json:"data"`
}

// Backfill CRUD methods

// CreateBackfill creates a new backfill for a sink consumer
func (c *Client) CreateBackfill(ctx context.Context, sinkIDOrName string, req *BackfillCreateRequest) (*BackfillResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/sinks/%s/backfills", sinkIDOrName), req)
	if err != nil {
		return nil, err
	}

	var result BackfillResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to create backfill: %w", err)
	}

	tflog.Info(ctx, "Created backfill", map[string]any{"id": result.ID, "sink": sinkIDOrName})
	return &result, nil
}

// GetBackfill retrieves a backfill by ID
func (c *Client) GetBackfill(ctx context.Context, sinkIDOrName string, backfillID string) (*BackfillResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/sinks/%s/backfills/%s", sinkIDOrName, backfillID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("backfill not found: %s", backfillID)
	}

	var result BackfillResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to get backfill: %w", err)
	}

	return &result, nil
}

// UpdateBackfill updates a backfill's state
func (c *Client) UpdateBackfill(ctx context.Context, sinkIDOrName string, backfillID string, req *BackfillUpdateRequest) (*BackfillResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/api/sinks/%s/backfills/%s", sinkIDOrName, backfillID), req)
	if err != nil {
		return nil, err
	}

	var result BackfillResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to update backfill: %w", err)
	}

	tflog.Info(ctx, "Updated backfill", map[string]any{"id": result.ID})
	return &result, nil
}

// DeleteBackfill deletes a backfill
func (c *Client) DeleteBackfill(ctx context.Context, sinkIDOrName string, backfillID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/sinks/%s/backfills/%s", sinkIDOrName, backfillID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Backfill already deleted", map[string]any{"id": backfillID})
		return nil
	}

	if err := c.handleResponse(ctx, resp, nil); err != nil {
		return fmt.Errorf("failed to delete backfill: %w", err)
	}

	tflog.Info(ctx, "Deleted backfill", map[string]any{"id": backfillID})
	return nil
}

// ListBackfills lists all backfills for a sink consumer
func (c *Client) ListBackfills(ctx context.Context, sinkIDOrName string) ([]BackfillResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/sinks/%s/backfills", sinkIDOrName), nil)
	if err != nil {
		return nil, err
	}

	var result BackfillListResponse
	if err := c.handleResponse(ctx, resp, &result); err != nil {
		return nil, fmt.Errorf("failed to list backfills: %w", err)
	}

	return result.Data, nil
}

// Error handling utilities

// IsNotFoundError checks if an error is a 404 Not Found error
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404")
}
