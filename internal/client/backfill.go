package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

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
