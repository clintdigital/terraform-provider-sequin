package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies expected interfaces
var (
	_ resource.Resource                = &BackfillResource{}
	_ resource.ResourceWithConfigure   = &BackfillResource{}
	_ resource.ResourceWithImportState = &BackfillResource{}
)

// BackfillResource defines the resource implementation
type BackfillResource struct {
	client *client.Client
}

// BackfillResourceModel describes the resource data model
type BackfillResourceModel struct {
	ID           types.String    `tfsdk:"id"`
	SinkConsumer types.String    `tfsdk:"sink_consumer"`
	Table        types.String    `tfsdk:"table"`
	State        types.String    `tfsdk:"state"`
	Status       *BackfillStatus `tfsdk:"status"`
}

// NewBackfillResource creates a new resource
func NewBackfillResource() resource.Resource {
	return &BackfillResource{}
}

// Metadata returns the resource type name
func (r *BackfillResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backfill"
}

// Schema defines the resource schema
func (r *BackfillResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a backfill operation that processes historical data through a sink consumer.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the backfill.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sink_consumer": schema.StringAttribute{
				Description: "Name or ID of the sink consumer to backfill.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"table": schema.StringAttribute{
				Description: "Source table in schema.table format (e.g. public.users). Required if the sink streams from multiple tables.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state": schema.StringAttribute{
				Description: "Desired state of the backfill: active or cancelled.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("active", "cancelled"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.SingleNestedAttribute{
				Description: "Current operational status of the backfill.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"state": schema.StringAttribute{
						Description: "Current state: active, completed, cancelled.",
						Computed:    true,
					},
					"inserted_at": schema.StringAttribute{
						Description: "ISO 8601 timestamp when the backfill was created.",
						Computed:    true,
					},
					"updated_at": schema.StringAttribute{
						Description: "ISO 8601 timestamp when the backfill was last updated.",
						Computed:    true,
					},
					"canceled_at": schema.StringAttribute{
						Description: "ISO 8601 timestamp when the backfill was cancelled.",
						Computed:    true,
					},
					"completed_at": schema.StringAttribute{
						Description: "ISO 8601 timestamp when the backfill completed.",
						Computed:    true,
					},
					"rows_ingested_count": schema.Int64Attribute{
						Description: "Number of rows delivered to the sink.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"rows_initial_count": schema.Int64Attribute{
						Description: "Total number of rows targeted for processing.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"rows_processed_count": schema.Int64Attribute{
						Description: "Number of rows examined during backfill.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"sort_column": schema.StringAttribute{
						Description: "Column used for ordering backfill data.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Configure adds the provider-configured client to the resource
func (r *BackfillResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a new backfill resource
func (r *BackfillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BackfillResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.BackfillCreateRequest{}
	if !data.Table.IsNull() && !data.Table.IsUnknown() {
		createReq.Table = data.Table.ValueString()
	}

	sinkConsumer := data.SinkConsumer.ValueString()
	created, err := r.client.CreateBackfill(ctx, sinkConsumer, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Backfill",
			"Could not create backfill: "+err.Error(),
		)
		return
	}

	mapBackfillResponseToModel(created, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Info(ctx, "Created backfill resource", map[string]any{"id": data.ID.ValueString()})
}

// Read refreshes the Terraform state with the latest data from the API
func (r *BackfillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BackfillResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backfillID := data.ID.ValueString()
	sinkConsumer := data.SinkConsumer.ValueString()

	backfill, err := r.client.GetBackfill(ctx, sinkConsumer, backfillID)
	if err != nil {
		if client.IsNotFoundError(err) {
			tflog.Warn(ctx, "Backfill not found, removing from state", map[string]any{"id": backfillID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Backfill",
			"Could not read backfill ID "+backfillID+": "+err.Error(),
		)
		return
	}

	mapBackfillResponseToModel(backfill, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the backfill state (e.g. cancel)
func (r *BackfillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackfillResourceModel
	var state BackfillResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backfillID := state.ID.ValueString()
	sinkConsumer := state.SinkConsumer.ValueString()

	updateReq := &client.BackfillUpdateRequest{}
	if !plan.State.IsNull() && !plan.State.IsUnknown() {
		updateReq.State = plan.State.ValueString()
	}

	updated, err := r.client.UpdateBackfill(ctx, sinkConsumer, backfillID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Backfill",
			"Could not update backfill ID "+backfillID+": "+err.Error(),
		)
		return
	}

	mapBackfillResponseToModel(updated, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Info(ctx, "Updated backfill resource", map[string]any{"id": backfillID})
}

// Delete deletes a backfill
func (r *BackfillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackfillResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backfillID := data.ID.ValueString()
	sinkConsumer := data.SinkConsumer.ValueString()

	err := r.client.DeleteBackfill(ctx, sinkConsumer, backfillID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Backfill",
			"Could not delete backfill ID "+backfillID+": "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted backfill", map[string]any{"id": backfillID})
}

// ImportState imports an existing backfill resource.
// Import format: <sink_consumer_name_or_id>/<backfill_id>
func (r *BackfillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: <sink_consumer>/<backfill_id>, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sink_consumer"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// mapBackfillResponseToModel maps the API response to the Terraform resource model
func mapBackfillResponseToModel(backfill *client.BackfillResponse, data *BackfillResourceModel) {
	data.ID = types.StringValue(backfill.ID)
	data.Table = types.StringValue(backfill.Table)
	data.State = types.StringValue(backfill.State)

	// Keep sink_consumer from state/plan (it's the user-provided name/ID used for API paths)
	// The API returns the consumer name in SinkConsumer field; use it if our field is empty
	if data.SinkConsumer.IsNull() || data.SinkConsumer.IsUnknown() || data.SinkConsumer.ValueString() == "" {
		data.SinkConsumer = types.StringValue(backfill.SinkConsumer)
	}

	data.Status = &BackfillStatus{
		State:              backfill.State,
		InsertedAt:         backfill.InsertedAt,
		UpdatedAt:          backfill.UpdatedAt,
		CanceledAt:         backfill.CanceledAt,
		CompletedAt:        backfill.CompletedAt,
		RowsIngestedCount:  backfill.RowsIngestedCount,
		RowsInitialCount:   backfill.RowsInitialCount,
		RowsProcessedCount: backfill.RowsProcessedCount,
		SortColumn:         backfill.SortColumn,
	}
}
