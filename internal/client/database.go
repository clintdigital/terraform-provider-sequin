package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

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
	Name             string            `json:"name"`
	URL              string            `json:"url,omitempty"`               // Alternative to individual connection params
	Hostname         string            `json:"hostname,omitempty"`
	Port             *int              `json:"port,omitempty"`
	Database         string            `json:"database,omitempty"`
	Username         string            `json:"username,omitempty"`
	Password         string            `json:"password,omitempty"`
	SSL              *bool             `json:"ssl,omitempty"`
	IPv6             *bool             `json:"ipv6,omitempty"`
	ReplicationSlots []ReplicationSlot `json:"replication_slots,omitempty"` // Required for create, optional for update
	Primary          *PrimaryDatabase  `json:"primary,omitempty"`           // For replica configuration
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
