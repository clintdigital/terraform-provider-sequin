package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SinkConsumerTable represents a table configuration in a sink consumer
type SinkConsumerTable struct {
	Name             string   `json:"name"`
	GroupColumnNames []string `json:"group_column_names,omitempty"`
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
	QueueURL       string `json:"queue_url,omitempty"`
	Region         string `json:"region,omitempty"`
	AccessKeyID    string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	IsFIFO         *bool  `json:"is_fifo,omitempty"`

	// Kinesis fields
	StreamARN string `json:"stream_arn,omitempty"`

	// Webhook fields
	HTTPEndpoint     string `json:"http_endpoint,omitempty"`
	HTTPEndpointPath string `json:"http_endpoint_path,omitempty"`
	Batch            *bool  `json:"batch,omitempty"`
}

// SinkConsumerRequest represents the request body for creating or updating a sink consumer
type SinkConsumerRequest struct {
	Name               string                  `json:"name"`
	Status             string                  `json:"status,omitempty"`                // active, disabled, paused
	Database           string                  `json:"database"`
	Source             *SinkConsumerSource     `json:"source,omitempty"`
	Tables             []SinkConsumerTable     `json:"tables"`
	Actions            []string                `json:"actions,omitempty"`                // insert, update, delete
	Destination        SinkConsumerDestination `json:"destination"`
	Filter             string                  `json:"filter,omitempty"`
	Transform          string                  `json:"transform,omitempty"`
	Enrichment         string                  `json:"enrichment,omitempty"`
	Routing            string                  `json:"routing,omitempty"`
	MessageGrouping    *bool                   `json:"message_grouping,omitempty"`
	BatchSize          *int                    `json:"batch_size,omitempty"`
	MaxRetryCount      *int                    `json:"max_retry_count,omitempty"`
	LoadSheddingPolicy string                  `json:"load_shedding_policy,omitempty"` // pause_on_full, discard_on_full
	TimestampFormat    string                  `json:"timestamp_format,omitempty"`     // iso8601, unix_microsecond
}

// SinkConsumerResponse represents a sink consumer resource from the API
type SinkConsumerResponse struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Status             string                  `json:"status"`
	Database           string                  `json:"database"`
	Source             *SinkConsumerSource     `json:"source,omitempty"`
	Tables             []SinkConsumerTable     `json:"tables"`
	Actions            []string                `json:"actions"`
	Destination        SinkConsumerDestination `json:"destination"`
	Filter             string                  `json:"filter,omitempty"`
	Transform          string                  `json:"transform,omitempty"`
	Enrichment         string                  `json:"enrichment,omitempty"`
	Routing            string                  `json:"routing,omitempty"`
	MessageGrouping    bool                    `json:"message_grouping"`
	BatchSize          int                     `json:"batch_size"`
	MaxRetryCount      *int                    `json:"max_retry_count,omitempty"`
	LoadSheddingPolicy string                  `json:"load_shedding_policy"`
	TimestampFormat    string                  `json:"timestamp_format"`
	StatusInfo         StatusResponse          `json:"status_info"`
}

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
