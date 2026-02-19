package resources

import (
	"context"
	"fmt"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies expected interfaces
var (
	_ resource.Resource                = &DatabaseResource{}
	_ resource.ResourceWithConfigure   = &DatabaseResource{}
	_ resource.ResourceWithImportState = &DatabaseResource{}
)

// DatabaseResource defines the resource implementation
type DatabaseResource struct {
	client *client.Client
}

// DatabaseResourceModel describes the resource data model
type DatabaseResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	URL              types.String `tfsdk:"url"`
	Hostname         types.String `tfsdk:"hostname"`
	Port             types.Int64  `tfsdk:"port"`
	Database         types.String `tfsdk:"database"`
	Username         types.String `tfsdk:"username"`
	Password         types.String `tfsdk:"password"`
	SSL              types.Bool   `tfsdk:"ssl"`
	IPv6             types.Bool   `tfsdk:"ipv6"`
	ReplicationSlots types.List   `tfsdk:"replication_slots"`
	Primary          types.Object `tfsdk:"primary"`
	// Computed fields
	UseLocalTunnel types.Bool  `tfsdk:"use_local_tunnel"`
	PoolSize       types.Int64 `tfsdk:"pool_size"`
	QueueInterval  types.Int64 `tfsdk:"queue_interval"`
	QueueTarget    types.Int64 `tfsdk:"queue_target"`
}

// NewDatabaseResource creates a new resource
func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

// Metadata returns the resource type name
func (r *DatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the resource schema
func (r *DatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PostgreSQL database connection in Sequin for streaming changes via CDC.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the database connection.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for the database connection.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "Full PostgreSQL connection URL (alternative to hostname/port/database/username/password).",
				Optional:    true,
				Sensitive:   true,
			},
			"hostname": schema.StringAttribute{
				Description: "Database server hostname.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Database server port (defaults to 5432).",
				Optional:    true,
				Computed:    true,
			},
			"database": schema.StringAttribute{
				Description: "Logical database name in PostgreSQL.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Database authentication username.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Database authentication password.",
				Optional:    true,
				Sensitive:   true,
			},
			"ssl": schema.BoolAttribute{
				Description: "Enable SSL for database connection (defaults to true).",
				Optional:    true,
				Computed:    true,
			},
			"ipv6": schema.BoolAttribute{
				Description: "Use IPv6 for database connection (defaults to false).",
				Optional:    true,
				Computed:    true,
			},
			"replication_slots": schema.ListNestedAttribute{
				Description: "Replication slot configuration (required for CDC).",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Replication slot ID (computed).",
							Computed:    true,
						},
						"publication_name": schema.StringAttribute{
							Description: "PostgreSQL publication name.",
							Required:    true,
						},
						"slot_name": schema.StringAttribute{
							Description: "PostgreSQL replication slot name.",
							Required:    true,
						},
						"status": schema.StringAttribute{
							Description: "Replication slot status: active, disabled.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"primary": schema.SingleNestedAttribute{
				Description: "Primary database configuration (for replica connections).",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Description: "Primary database hostname.",
						Required:    true,
					},
					"database": schema.StringAttribute{
						Description: "Primary database name.",
						Required:    true,
					},
					"username": schema.StringAttribute{
						Description: "Primary database username.",
						Required:    true,
					},
					"password": schema.StringAttribute{
						Description: "Primary database password.",
						Required:    true,
						Sensitive:   true,
					},
					"port": schema.Int64Attribute{
						Description: "Primary database port.",
						Optional:    true,
					},
					"ssl": schema.BoolAttribute{
						Description: "Enable SSL for primary connection.",
						Optional:    true,
					},
				},
			},
			// Computed fields
			"use_local_tunnel": schema.BoolAttribute{
				Description: "Whether a local tunnel is being used for connection.",
				Computed:    true,
			},
			"pool_size": schema.Int64Attribute{
				Description: "Connection pool size.",
				Computed:    true,
			},
			"queue_interval": schema.Int64Attribute{
				Description: "Queue processing interval.",
				Computed:    true,
			},
			"queue_target": schema.Int64Attribute{
				Description: "Queue processing target.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider-configured client to the resource
func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates a new database resource
func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatabaseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	createReq := &client.DatabaseRequest{
		Name: data.Name.ValueString(),
	}

	// Connection details (URL or individual params)
	if !data.URL.IsNull() {
		createReq.URL = data.URL.ValueString()
	}
	if !data.Hostname.IsNull() {
		createReq.Hostname = data.Hostname.ValueString()
	}
	if !data.Port.IsNull() {
		port := int(data.Port.ValueInt64())
		createReq.Port = &port
	}
	if !data.Database.IsNull() {
		createReq.Database = data.Database.ValueString()
	}
	if !data.Username.IsNull() {
		createReq.Username = data.Username.ValueString()
	}
	if !data.Password.IsNull() {
		createReq.Password = data.Password.ValueString()
	}
	if !data.SSL.IsNull() {
		ssl := data.SSL.ValueBool()
		createReq.SSL = &ssl
	}
	if !data.IPv6.IsNull() {
		ipv6 := data.IPv6.ValueBool()
		createReq.IPv6 = &ipv6
	}

	// Parse replication slots
	var slotsData []struct {
		ID              types.String `tfsdk:"id"`
		PublicationName types.String `tfsdk:"publication_name"`
		SlotName        types.String `tfsdk:"slot_name"`
		Status          types.String `tfsdk:"status"`
	}
	resp.Diagnostics.Append(data.ReplicationSlots.ElementsAs(ctx, &slotsData, false)...)

	createReq.ReplicationSlots = make([]client.ReplicationSlot, len(slotsData))
	for i, slot := range slotsData {
		createReq.ReplicationSlots[i].PublicationName = slot.PublicationName.ValueString()
		createReq.ReplicationSlots[i].SlotName = slot.SlotName.ValueString()
		if !slot.Status.IsNull() {
			createReq.ReplicationSlots[i].Status = slot.Status.ValueString()
		}
	}

	// Parse primary database if provided
	if !data.Primary.IsNull() {
		primary := &client.PrimaryDatabase{}
		primaryAttrs := data.Primary.Attributes()

		primary.Hostname = primaryAttrs["hostname"].(types.String).ValueString()
		primary.Database = primaryAttrs["database"].(types.String).ValueString()
		primary.Username = primaryAttrs["username"].(types.String).ValueString()
		primary.Password = primaryAttrs["password"].(types.String).ValueString()

		if port, ok := primaryAttrs["port"].(types.Int64); ok && !port.IsNull() {
			p := int(port.ValueInt64())
			primary.Port = &p
		}
		if ssl, ok := primaryAttrs["ssl"].(types.Bool); ok && !ssl.IsNull() {
			s := ssl.ValueBool()
			primary.SSL = &s
		}

		createReq.Primary = primary
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	created, err := r.client.CreateDatabase(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Database",
			"Could not create database: "+err.Error(),
		)
		return
	}

	// Map response to model
	r.mapResponseToModel(ctx, created, &data, &resp.Diagnostics)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Created database resource", map[string]any{"id": data.ID.ValueString()})
}

// Read refreshes the Terraform state with the latest data from the API
func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatabaseResourceModel

	// Read Terraform current state into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	dbID := data.ID.ValueString()
	database, err := r.client.GetDatabase(ctx, dbID)
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource was deleted outside Terraform
			tflog.Warn(ctx, "Database not found, removing from state", map[string]any{"id": dbID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Database",
			"Could not read database ID "+dbID+": "+err.Error(),
		)
		return
	}

	// Update model with latest values from API (drift detection)
	r.mapResponseToModel(ctx, database, &data, &resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates an existing database resource
func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DatabaseResourceModel

	// Read Terraform plan and current state
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request (same structure as create)
	updateReq := &client.DatabaseRequest{
		Name: plan.Name.ValueString(),
	}

	// Connection details
	if !plan.URL.IsNull() {
		updateReq.URL = plan.URL.ValueString()
	}
	if !plan.Hostname.IsNull() {
		updateReq.Hostname = plan.Hostname.ValueString()
	}
	if !plan.Port.IsNull() {
		port := int(plan.Port.ValueInt64())
		updateReq.Port = &port
	}
	if !plan.Database.IsNull() {
		updateReq.Database = plan.Database.ValueString()
	}
	if !plan.Username.IsNull() {
		updateReq.Username = plan.Username.ValueString()
	}
	if !plan.Password.IsNull() {
		updateReq.Password = plan.Password.ValueString()
	}
	if !plan.SSL.IsNull() {
		ssl := plan.SSL.ValueBool()
		updateReq.SSL = &ssl
	}
	if !plan.IPv6.IsNull() {
		ipv6 := plan.IPv6.ValueBool()
		updateReq.IPv6 = &ipv6
	}

	// Parse replication slots (for update, include ID)
	var slotsData []struct {
		ID              types.String `tfsdk:"id"`
		PublicationName types.String `tfsdk:"publication_name"`
		SlotName        types.String `tfsdk:"slot_name"`
		Status          types.String `tfsdk:"status"`
	}
	resp.Diagnostics.Append(plan.ReplicationSlots.ElementsAs(ctx, &slotsData, false)...)

	updateReq.ReplicationSlots = make([]client.ReplicationSlot, len(slotsData))
	for i, slot := range slotsData {
		if !slot.ID.IsNull() {
			updateReq.ReplicationSlots[i].ID = slot.ID.ValueString()
		}
		updateReq.ReplicationSlots[i].PublicationName = slot.PublicationName.ValueString()
		updateReq.ReplicationSlots[i].SlotName = slot.SlotName.ValueString()
		if !slot.Status.IsNull() {
			updateReq.ReplicationSlots[i].Status = slot.Status.ValueString()
		}
	}

	// Parse primary database if provided
	if !plan.Primary.IsNull() {
		primary := &client.PrimaryDatabase{}
		primaryAttrs := plan.Primary.Attributes()

		primary.Hostname = primaryAttrs["hostname"].(types.String).ValueString()
		primary.Database = primaryAttrs["database"].(types.String).ValueString()
		primary.Username = primaryAttrs["username"].(types.String).ValueString()
		primary.Password = primaryAttrs["password"].(types.String).ValueString()

		if port, ok := primaryAttrs["port"].(types.Int64); ok && !port.IsNull() {
			p := int(port.ValueInt64())
			primary.Port = &p
		}
		if ssl, ok := primaryAttrs["ssl"].(types.Bool); ok && !ssl.IsNull() {
			s := ssl.ValueBool()
			primary.SSL = &s
		}

		updateReq.Primary = primary
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	dbID := state.ID.ValueString()
	updated, err := r.client.UpdateDatabase(ctx, dbID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Database",
			"Could not update database ID "+dbID+": "+err.Error(),
		)
		return
	}

	// Update model with response
	r.mapResponseToModel(ctx, updated, &plan, &resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Info(ctx, "Updated database resource", map[string]any{"id": dbID})
}

// Delete deletes a database resource
func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DatabaseResourceModel

	// Read Terraform current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete
	dbID := data.ID.ValueString()
	err := r.client.DeleteDatabase(ctx, dbID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Database",
			"Could not delete database ID "+dbID+": "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted database resource", map[string]any{"id": dbID})
	// State is automatically removed by Terraform after successful Delete
}

// ImportState imports an existing database resource by ID
func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID: terraform import sequin_database.example <database-id>
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToModel maps API response to Terraform model
func (r *DatabaseResource) mapResponseToModel(ctx context.Context, response *client.DatabaseResponse, model *DatabaseResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(response.ID)
	model.Name = types.StringValue(response.Name)
	model.Hostname = types.StringValue(response.Hostname)
	model.Port = types.Int64Value(int64(response.Port))
	model.Database = types.StringValue(response.Database)
	model.Username = types.StringValue(response.Username)
	// Password is obfuscated in response, keep from state
	model.SSL = types.BoolValue(response.SSL)
	model.IPv6 = types.BoolValue(response.IPv6)

	// Computed fields
	model.UseLocalTunnel = types.BoolValue(response.UseLocalTunnel)
	model.PoolSize = types.Int64Value(int64(response.PoolSize))
	model.QueueInterval = types.Int64Value(int64(response.QueueInterval))
	model.QueueTarget = types.Int64Value(int64(response.QueueTarget))

	// Map replication slots
	slotsList := make([]attr.Value, len(response.ReplicationSlots))
	for i, slot := range response.ReplicationSlots {
		slotAttrs := map[string]attr.Value{
			"id":               types.StringValue(slot.ID),
			"publication_name": types.StringValue(slot.PublicationName),
			"slot_name":        types.StringValue(slot.SlotName),
		}

		if slot.Status != "" {
			slotAttrs["status"] = types.StringValue(slot.Status)
		} else {
			slotAttrs["status"] = types.StringNull()
		}

		obj, d := types.ObjectValue(map[string]attr.Type{
			"id":               types.StringType,
			"publication_name": types.StringType,
			"slot_name":        types.StringType,
			"status":           types.StringType,
		}, slotAttrs)
		diags.Append(d...)
		slotsList[i] = obj
	}
	list, d := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":               types.StringType,
			"publication_name": types.StringType,
			"slot_name":        types.StringType,
			"status":           types.StringType,
		},
	}, slotsList)
	diags.Append(d...)
	model.ReplicationSlots = list

	// Map primary database if present
	if response.Primary != nil {
		primaryAttrs := map[string]attr.Value{
			"hostname": types.StringValue(response.Primary.Hostname),
			"database": types.StringValue(response.Primary.Database),
			"username": types.StringValue(response.Primary.Username),
			"password": types.StringValue(response.Primary.Password),
		}

		if response.Primary.Port != nil {
			primaryAttrs["port"] = types.Int64Value(int64(*response.Primary.Port))
		} else {
			primaryAttrs["port"] = types.Int64Null()
		}

		if response.Primary.SSL != nil {
			primaryAttrs["ssl"] = types.BoolValue(*response.Primary.SSL)
		} else {
			primaryAttrs["ssl"] = types.BoolNull()
		}

		obj, d := types.ObjectValue(map[string]attr.Type{
			"hostname": types.StringType,
			"database": types.StringType,
			"username": types.StringType,
			"password": types.StringType,
			"port":     types.Int64Type,
			"ssl":      types.BoolType,
		}, primaryAttrs)
		diags.Append(d...)
		model.Primary = obj
	} else {
		model.Primary = types.ObjectNull(map[string]attr.Type{
			"hostname": types.StringType,
			"database": types.StringType,
			"username": types.StringType,
			"password": types.StringType,
			"port":     types.Int64Type,
			"ssl":      types.BoolType,
		})
	}

}
