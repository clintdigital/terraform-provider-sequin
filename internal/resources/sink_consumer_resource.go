package resources

import (
	"context"
	"fmt"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies expected interfaces
var (
	_ resource.Resource                = &SinkConsumerResource{}
	_ resource.ResourceWithConfigure   = &SinkConsumerResource{}
	_ resource.ResourceWithImportState = &SinkConsumerResource{}
)

// SinkConsumerResource defines the resource implementation
type SinkConsumerResource struct {
	client *client.Client
}

// SinkConsumerResourceModel describes the resource data model
type SinkConsumerResourceModel struct {
	ID                 types.String    `tfsdk:"id"`
	Name               types.String    `tfsdk:"name"`
	Status             types.String    `tfsdk:"status"`
	Database           types.String    `tfsdk:"database"`
	Source             types.Object    `tfsdk:"source"`
	Tables             types.List      `tfsdk:"tables"`
	Actions            types.List      `tfsdk:"actions"`
	Destination        types.Object    `tfsdk:"destination"`
	Filter             types.String    `tfsdk:"filter"`
	Transform          types.String    `tfsdk:"transform"`
	Enrichment         types.String    `tfsdk:"enrichment"`
	Routing            types.String `tfsdk:"routing"`
	MessageGrouping    types.Bool   `tfsdk:"message_grouping"`
	BatchSize          types.Int64  `tfsdk:"batch_size"`
	MaxRetryCount      types.Int64  `tfsdk:"max_retry_count"`
	LoadSheddingPolicy types.String `tfsdk:"load_shedding_policy"`
	TimestampFormat    types.String `tfsdk:"timestamp_format"`
	StatusInfo         types.Object `tfsdk:"status_info"`
}

// NewSinkConsumerResource creates a new resource
func NewSinkConsumerResource() resource.Resource {
	return &SinkConsumerResource{}
}

// Metadata returns the resource type name
func (r *SinkConsumerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sink_consumer"
}

// Schema defines the resource schema
func (r *SinkConsumerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a sink consumer that streams database changes to Kafka, SQS, Kinesis, or webhook endpoints.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the sink consumer.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for the sink consumer.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Desired status of the sink consumer: active, disabled, paused.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("active", "disabled", "paused"),
				},
			},
			"database": schema.StringAttribute{
				Description: "ID of the database connection to stream from.",
				Required:    true,
			},
			"source": schema.SingleNestedAttribute{
				Description: "Source configuration for filtering schemas and tables.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"include_schemas": schema.ListAttribute{
						Description: "List of schema names to include (e.g. ['public']).",
						Optional:    true,
						ElementType: types.StringType,
					},
					"exclude_schemas": schema.ListAttribute{
						Description: "List of schema names to exclude.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"include_tables": schema.ListAttribute{
						Description: "List of table names to include.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"exclude_tables": schema.ListAttribute{
						Description: "List of table names to exclude.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"tables": schema.ListNestedAttribute{
				Description: "List of tables to stream changes from.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Table name (can be schema-qualified like 'public.users').",
							Required:    true,
						},
						"group_column_names": schema.ListAttribute{
							Description: "Column names to use for message grouping/ordering.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"actions": schema.ListAttribute{
				Description: "List of change actions to capture: insert, update, delete.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"destination": schema.SingleNestedAttribute{
				Description: "Destination configuration for where to send changes.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Destination type: kafka, sqs, kinesis, webhook.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("kafka", "sqs", "kinesis", "webhook"),
						},
					},
					// Kafka fields
					"hosts": schema.StringAttribute{
						Description: "Kafka broker hosts (comma-separated).",
						Optional:    true,
					},
					"topic": schema.StringAttribute{
						Description: "Kafka topic name.",
						Optional:    true,
					},
					"tls": schema.BoolAttribute{
						Description: "Enable TLS for Kafka connection.",
						Optional:    true,
					},
					"username": schema.StringAttribute{
						Description: "Username for Kafka authentication.",
						Optional:    true,
					},
					"password": schema.StringAttribute{
						Description: "Password for Kafka authentication.",
						Optional:    true,
						Sensitive:   true,
					},
					"sasl_mechanism": schema.StringAttribute{
						Description: "SASL mechanism: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, AWS_MSK_IAM.",
						Optional:    true,
					},
					"aws_region": schema.StringAttribute{
						Description: "AWS region for MSK IAM authentication.",
						Optional:    true,
					},
					"aws_access_key_id": schema.StringAttribute{
						Description: "AWS access key ID for MSK IAM authentication.",
						Optional:    true,
						Sensitive:   true,
					},
					"aws_secret_access_key": schema.StringAttribute{
						Description: "AWS secret access key for MSK IAM authentication.",
						Optional:    true,
						Sensitive:   true,
					},
					// SQS fields
					"queue_url": schema.StringAttribute{
						Description: "SQS queue URL.",
						Optional:    true,
					},
					"region": schema.StringAttribute{
						Description: "AWS region for SQS/Kinesis.",
						Optional:    true,
					},
					"access_key_id": schema.StringAttribute{
						Description: "AWS access key ID.",
						Optional:    true,
						Sensitive:   true,
					},
					"secret_access_key": schema.StringAttribute{
						Description: "AWS secret access key.",
						Optional:    true,
						Sensitive:   true,
					},
					"is_fifo": schema.BoolAttribute{
						Description: "Whether the SQS queue is FIFO.",
						Optional:    true,
					},
					// Kinesis fields
					"stream_arn": schema.StringAttribute{
						Description: "Kinesis stream ARN.",
						Optional:    true,
					},
					// Webhook fields
					"http_endpoint": schema.StringAttribute{
						Description: "Webhook HTTP endpoint base URL.",
						Optional:    true,
					},
					"http_endpoint_path": schema.StringAttribute{
						Description: "Webhook HTTP endpoint path.",
						Optional:    true,
					},
					"batch": schema.BoolAttribute{
						Description: "Enable batched delivery for webhooks.",
						Optional:    true,
					},
				},
			},
			"filter": schema.StringAttribute{
				Description: "Named filter function to control which rows trigger changes.",
				Optional:    true,
			},
			"transform": schema.StringAttribute{
				Description: "Named transform function to reshape messages before delivery.",
				Optional:    true,
			},
			"enrichment": schema.StringAttribute{
				Description: "Named enrichment function that runs a SQL query to add data to messages.",
				Optional:    true,
			},
			"routing": schema.StringAttribute{
				Description: "Named routing function to dynamically direct messages to destinations.",
				Optional:    true,
			},
			"message_grouping": schema.BoolAttribute{
				Description: "Enable message grouping for ordered delivery.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"batch_size": schema.Int64Attribute{
				Description: "Number of messages to batch together.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_retry_count": schema.Int64Attribute{
				Description: "Maximum number of retry attempts for failed deliveries.",
				Optional:    true,
			},
			"load_shedding_policy": schema.StringAttribute{
				Description: "Policy for handling overload: pause_on_full, discard_on_full.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("pause_on_full", "discard_on_full"),
				},
			},
			"timestamp_format": schema.StringAttribute{
				Description: "Format for timestamps: iso8601, unix_microsecond.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("iso8601", "unix_microsecond"),
				},
			},
			"status_info": schema.SingleNestedAttribute{
				Description: "Current operational status of the sink consumer.",
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"state": schema.StringAttribute{
						Description: "Current state: active, pending, failed, disabled.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "ISO 8601 timestamp when the resource was created.",
						Computed:    true,
					},
					"updated_at": schema.StringAttribute{
						Description: "ISO 8601 timestamp of the last update.",
						Computed:    true,
					},
					"last_error": schema.StringAttribute{
						Description: "Most recent error message if any.",
						Computed:    true,
					},
				},
			},
		},
	}
}

// Configure adds the provider-configured client to the resource
func (r *SinkConsumerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a new sink consumer resource
func (r *SinkConsumerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SinkConsumerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	createReq := &client.SinkConsumerRequest{
		Name:     data.Name.ValueString(),
		Database: data.Database.ValueString(),
	}

	// Optional fields
	if !data.Status.IsNull() {
		createReq.Status = data.Status.ValueString()
	}

	// Parse source
	if !data.Source.IsNull() {
		source := &client.SinkConsumerSource{}
		sourceAttrs := data.Source.Attributes()

		if includeSchemas, ok := sourceAttrs["include_schemas"].(types.List); ok && !includeSchemas.IsNull() {
			   var schemas []string
			   resp.Diagnostics.Append(includeSchemas.ElementsAs(ctx, &schemas, false)...)
			   source.IncludeSchemas = schemas
			   if excludeSchemas, ok := sourceAttrs["exclude_schemas"].(types.List); ok && !excludeSchemas.IsNull() {
				   var schemas []string
				   resp.Diagnostics.Append(excludeSchemas.ElementsAs(ctx, &schemas, false)...)
				   source.ExcludeSchemas = schemas
			   }
		}
		if includeTables, ok := sourceAttrs["include_tables"].(types.List); ok && !includeTables.IsNull() {
			var tables []string
			resp.Diagnostics.Append(includeTables.ElementsAs(ctx, &tables, false)...)
			source.IncludeTables = tables
		}
		if excludeTables, ok := sourceAttrs["exclude_tables"].(types.List); ok && !excludeTables.IsNull() {
			var tables []string
			resp.Diagnostics.Append(excludeTables.ElementsAs(ctx, &tables, false)...)
			source.ExcludeTables = tables
		}

		createReq.Source = source
	}

	// Parse tables
	var tablesData []struct {
		Name              types.String `tfsdk:"name"`
		GroupColumnNames  types.List   `tfsdk:"group_column_names"`
	}
	resp.Diagnostics.Append(data.Tables.ElementsAs(ctx, &tablesData, false)...)

	createReq.Tables = make([]client.SinkConsumerTable, len(tablesData))
	for i, table := range tablesData {
		createReq.Tables[i].Name = table.Name.ValueString()
		if !table.GroupColumnNames.IsNull() {
			var groupCols []string
			resp.Diagnostics.Append(table.GroupColumnNames.ElementsAs(ctx, &groupCols, false)...)
			createReq.Tables[i].GroupColumnNames = groupCols
		}
	}

	// Parse actions
	if !data.Actions.IsNull() {
		var actions []string
		resp.Diagnostics.Append(data.Actions.ElementsAs(ctx, &actions, false)...)
		createReq.Actions = actions
	}

	// Parse destination
	destAttrs := data.Destination.Attributes()
	createReq.Destination = client.SinkConsumerDestination{
		Type: destAttrs["type"].(types.String).ValueString(),
	}

	// Kafka fields
	if hosts, ok := destAttrs["hosts"].(types.String); ok && !hosts.IsNull() {
		createReq.Destination.Hosts = hosts.ValueString()
	}
	if topic, ok := destAttrs["topic"].(types.String); ok && !topic.IsNull() {
		createReq.Destination.Topic = topic.ValueString()
	}
	if tls, ok := destAttrs["tls"].(types.Bool); ok && !tls.IsNull() {
		val := tls.ValueBool()
		createReq.Destination.TLS = &val
	}
	if username, ok := destAttrs["username"].(types.String); ok && !username.IsNull() {
		createReq.Destination.Username = username.ValueString()
	}
	if password, ok := destAttrs["password"].(types.String); ok && !password.IsNull() {
		createReq.Destination.Password = password.ValueString()
	}
	if saslMech, ok := destAttrs["sasl_mechanism"].(types.String); ok && !saslMech.IsNull() {
		createReq.Destination.SASLMechanism = saslMech.ValueString()
	}
	if awsRegion, ok := destAttrs["aws_region"].(types.String); ok && !awsRegion.IsNull() {
		createReq.Destination.AWSRegion = awsRegion.ValueString()
	}
	if awsAccessKey, ok := destAttrs["aws_access_key_id"].(types.String); ok && !awsAccessKey.IsNull() {
		createReq.Destination.AWSAccessKeyID = awsAccessKey.ValueString()
	}
	if awsSecretKey, ok := destAttrs["aws_secret_access_key"].(types.String); ok && !awsSecretKey.IsNull() {
		createReq.Destination.AWSSecretAccessKey = awsSecretKey.ValueString()
	}

	// SQS fields
	if queueURL, ok := destAttrs["queue_url"].(types.String); ok && !queueURL.IsNull() {
		createReq.Destination.QueueURL = queueURL.ValueString()
	}
	if region, ok := destAttrs["region"].(types.String); ok && !region.IsNull() {
		createReq.Destination.Region = region.ValueString()
	}
	if accessKey, ok := destAttrs["access_key_id"].(types.String); ok && !accessKey.IsNull() {
		createReq.Destination.AccessKeyID = accessKey.ValueString()
	}
	if secretKey, ok := destAttrs["secret_access_key"].(types.String); ok && !secretKey.IsNull() {
		createReq.Destination.SecretAccessKey = secretKey.ValueString()
	}
	if isFIFO, ok := destAttrs["is_fifo"].(types.Bool); ok && !isFIFO.IsNull() {
		val := isFIFO.ValueBool()
		createReq.Destination.IsFIFO = &val
	}

	// Kinesis fields
	if streamARN, ok := destAttrs["stream_arn"].(types.String); ok && !streamARN.IsNull() {
		createReq.Destination.StreamARN = streamARN.ValueString()
	}

	// Webhook fields
	if httpEndpoint, ok := destAttrs["http_endpoint"].(types.String); ok && !httpEndpoint.IsNull() {
		createReq.Destination.HTTPEndpoint = httpEndpoint.ValueString()
	}
	if httpEndpointPath, ok := destAttrs["http_endpoint_path"].(types.String); ok && !httpEndpointPath.IsNull() {
		createReq.Destination.HTTPEndpointPath = httpEndpointPath.ValueString()
	}
	if batch, ok := destAttrs["batch"].(types.Bool); ok && !batch.IsNull() {
		val := batch.ValueBool()
		createReq.Destination.Batch = &val
	}

	// Optional string fields
	if !data.Filter.IsNull() {
		createReq.Filter = data.Filter.ValueString()
	}
	if !data.Transform.IsNull() {
		createReq.Transform = data.Transform.ValueString()
	}
	if !data.Enrichment.IsNull() {
		createReq.Enrichment = data.Enrichment.ValueString()
	}
	if !data.Routing.IsNull() {
		createReq.Routing = data.Routing.ValueString()
	}
	if !data.LoadSheddingPolicy.IsNull() {
		createReq.LoadSheddingPolicy = data.LoadSheddingPolicy.ValueString()
	}
	if !data.TimestampFormat.IsNull() {
		createReq.TimestampFormat = data.TimestampFormat.ValueString()
	}

	// Optional bool/int fields
	if !data.MessageGrouping.IsNull() {
		val := data.MessageGrouping.ValueBool()
		createReq.MessageGrouping = &val
	}
	if !data.BatchSize.IsNull() {
		val := int(data.BatchSize.ValueInt64())
		createReq.BatchSize = &val
	}
	if !data.MaxRetryCount.IsNull() {
		val := int(data.MaxRetryCount.ValueInt64())
		createReq.MaxRetryCount = &val
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Store original null states and sensitive values from plan
	sourceWasNull := data.Source.IsNull()
	transformWasNull := data.Transform.IsNull()
	enrichmentWasNull := data.Enrichment.IsNull()
	originalDestination := data.Destination

	// Call API
	created, err := r.client.CreateSinkConsumer(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Sink Consumer",
			"Could not create sink consumer: "+err.Error(),
		)
		return
	}

	// Map response to model (this will overwrite destination)
	r.mapResponseToModel(ctx, created, &data, &resp.Diagnostics)

	// Restore destination from plan to preserve sensitive values
	data.Destination = originalDestination

	// Restore null states if they were null in plan
	if sourceWasNull {
		data.Source = types.ObjectNull(map[string]attr.Type{
			"include_schemas": types.ListType{ElemType: types.StringType},
			"exclude_schemas": types.ListType{ElemType: types.StringType},
			"include_tables":  types.ListType{ElemType: types.StringType},
			"exclude_tables":  types.ListType{ElemType: types.StringType},
		})
	}
	if transformWasNull {
		data.Transform = types.StringNull()
	}
	if enrichmentWasNull {
		data.Enrichment = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Created sink consumer resource", map[string]any{"id": data.ID.ValueString()})
}

// Read refreshes the Terraform state with the latest data from the API
func (r *SinkConsumerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SinkConsumerResourceModel

	// Read Terraform current state into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state from API
	consumerID := data.ID.ValueString()
	consumer, err := r.client.GetSinkConsumer(ctx, consumerID)
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource was deleted outside Terraform
			tflog.Warn(ctx, "Sink consumer not found, removing from state", map[string]any{"id": consumerID})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Sink Consumer",
			"Could not read sink consumer ID "+consumerID+": "+err.Error(),
		)
		return
	}

	// Update model with latest values from API (drift detection)
	r.mapResponseToModel(ctx, consumer, &data, &resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates an existing sink consumer resource
func (r *SinkConsumerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SinkConsumerResourceModel

	// Read Terraform plan and current state
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request (same structure as create)
	updateReq := &client.SinkConsumerRequest{
		Name:     plan.Name.ValueString(),
		Database: plan.Database.ValueString(),
	}

	// Copy all the same logic from Create for building the request
	if !plan.Status.IsNull() {
		updateReq.Status = plan.Status.ValueString()
	}

	// Parse source
	if !plan.Source.IsNull() {
		source := &client.SinkConsumerSource{}
		sourceAttrs := plan.Source.Attributes()

		if includeSchemas, ok := sourceAttrs["include_schemas"].(types.List); ok && !includeSchemas.IsNull() {
			var schemas []string
			resp.Diagnostics.Append(includeSchemas.ElementsAs(ctx, &schemas, false)...)
			source.IncludeSchemas = schemas
		}
		if excludeSchemas, ok := sourceAttrs["exclude_schemas"].(types.List); ok && !excludeSchemas.IsNull() {
			var schemas []string
			resp.Diagnostics.Append(excludeSchemas.ElementsAs(ctx, &schemas, false)...)
			source.ExcludeSchemas = schemas
		}
		if includeTables, ok := sourceAttrs["include_tables"].(types.List); ok && !includeTables.IsNull() {
			var tables []string
			resp.Diagnostics.Append(includeTables.ElementsAs(ctx, &tables, false)...)
			source.IncludeTables = tables
		}
		if excludeTables, ok := sourceAttrs["exclude_tables"].(types.List); ok && !excludeTables.IsNull() {
			var tables []string
			resp.Diagnostics.Append(excludeTables.ElementsAs(ctx, &tables, false)...)
			source.ExcludeTables = tables
		}

		updateReq.Source = source
	}

	// Parse tables
	var tablesData []struct {
		Name              types.String `tfsdk:"name"`
		GroupColumnNames  types.List   `tfsdk:"group_column_names"`
	}
	resp.Diagnostics.Append(plan.Tables.ElementsAs(ctx, &tablesData, false)...)

	updateReq.Tables = make([]client.SinkConsumerTable, len(tablesData))
	for i, table := range tablesData {
		updateReq.Tables[i].Name = table.Name.ValueString()
		if !table.GroupColumnNames.IsNull() {
			var groupCols []string
			resp.Diagnostics.Append(table.GroupColumnNames.ElementsAs(ctx, &groupCols, false)...)
			updateReq.Tables[i].GroupColumnNames = groupCols
		}
	}

	// Parse actions
	if !plan.Actions.IsNull() {
		var actions []string
		resp.Diagnostics.Append(plan.Actions.ElementsAs(ctx, &actions, false)...)
		updateReq.Actions = actions
	}

	// Parse destination
	destAttrs := plan.Destination.Attributes()
	updateReq.Destination = client.SinkConsumerDestination{
		Type: destAttrs["type"].(types.String).ValueString(),
	}

	// Kafka fields
	if hosts, ok := destAttrs["hosts"].(types.String); ok && !hosts.IsNull() {
		updateReq.Destination.Hosts = hosts.ValueString()
	}
	if topic, ok := destAttrs["topic"].(types.String); ok && !topic.IsNull() {
		updateReq.Destination.Topic = topic.ValueString()
	}
	if tls, ok := destAttrs["tls"].(types.Bool); ok && !tls.IsNull() {
		val := tls.ValueBool()
		updateReq.Destination.TLS = &val
	}
	if username, ok := destAttrs["username"].(types.String); ok && !username.IsNull() {
		updateReq.Destination.Username = username.ValueString()
	}
	if password, ok := destAttrs["password"].(types.String); ok && !password.IsNull() {
		updateReq.Destination.Password = password.ValueString()
	}
	if saslMech, ok := destAttrs["sasl_mechanism"].(types.String); ok && !saslMech.IsNull() {
		updateReq.Destination.SASLMechanism = saslMech.ValueString()
	}
	if awsRegion, ok := destAttrs["aws_region"].(types.String); ok && !awsRegion.IsNull() {
		updateReq.Destination.AWSRegion = awsRegion.ValueString()
	}
	if awsAccessKey, ok := destAttrs["aws_access_key_id"].(types.String); ok && !awsAccessKey.IsNull() {
		updateReq.Destination.AWSAccessKeyID = awsAccessKey.ValueString()
	}
	if awsSecretKey, ok := destAttrs["aws_secret_access_key"].(types.String); ok && !awsSecretKey.IsNull() {
		updateReq.Destination.AWSSecretAccessKey = awsSecretKey.ValueString()
	}

	// SQS fields
	if queueURL, ok := destAttrs["queue_url"].(types.String); ok && !queueURL.IsNull() {
		updateReq.Destination.QueueURL = queueURL.ValueString()
	}
	if region, ok := destAttrs["region"].(types.String); ok && !region.IsNull() {
		updateReq.Destination.Region = region.ValueString()
	}
	if accessKey, ok := destAttrs["access_key_id"].(types.String); ok && !accessKey.IsNull() {
		updateReq.Destination.AccessKeyID = accessKey.ValueString()
	}
	if secretKey, ok := destAttrs["secret_access_key"].(types.String); ok && !secretKey.IsNull() {
		updateReq.Destination.SecretAccessKey = secretKey.ValueString()
	}
	if isFIFO, ok := destAttrs["is_fifo"].(types.Bool); ok && !isFIFO.IsNull() {
		val := isFIFO.ValueBool()
		updateReq.Destination.IsFIFO = &val
	}

	// Kinesis fields
	if streamARN, ok := destAttrs["stream_arn"].(types.String); ok && !streamARN.IsNull() {
		updateReq.Destination.StreamARN = streamARN.ValueString()
	}

	// Webhook fields
	if httpEndpoint, ok := destAttrs["http_endpoint"].(types.String); ok && !httpEndpoint.IsNull() {
		updateReq.Destination.HTTPEndpoint = httpEndpoint.ValueString()
	}
	if httpEndpointPath, ok := destAttrs["http_endpoint_path"].(types.String); ok && !httpEndpointPath.IsNull() {
		updateReq.Destination.HTTPEndpointPath = httpEndpointPath.ValueString()
	}
	if batch, ok := destAttrs["batch"].(types.Bool); ok && !batch.IsNull() {
		val := batch.ValueBool()
		updateReq.Destination.Batch = &val
	}

	// Optional string fields
	if !plan.Filter.IsNull() {
		updateReq.Filter = plan.Filter.ValueString()
	}
	if !plan.Transform.IsNull() {
		updateReq.Transform = plan.Transform.ValueString()
	}
	if !plan.Enrichment.IsNull() {
		updateReq.Enrichment = plan.Enrichment.ValueString()
	}
	if !plan.Routing.IsNull() {
		updateReq.Routing = plan.Routing.ValueString()
	}
	if !plan.LoadSheddingPolicy.IsNull() {
		updateReq.LoadSheddingPolicy = plan.LoadSheddingPolicy.ValueString()
	}
	if !plan.TimestampFormat.IsNull() {
		updateReq.TimestampFormat = plan.TimestampFormat.ValueString()
	}

	// Optional bool/int fields
	if !plan.MessageGrouping.IsNull() {
		val := plan.MessageGrouping.ValueBool()
		updateReq.MessageGrouping = &val
	}
	if !plan.BatchSize.IsNull() {
		val := int(plan.BatchSize.ValueInt64())
		updateReq.BatchSize = &val
	}
	if !plan.MaxRetryCount.IsNull() {
		val := int(plan.MaxRetryCount.ValueInt64())
		updateReq.MaxRetryCount = &val
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	consumerID := state.ID.ValueString()
	updated, err := r.client.UpdateSinkConsumer(ctx, consumerID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Sink Consumer",
			"Could not update sink consumer ID "+consumerID+": "+err.Error(),
		)
		return
	}

	// Update model with response
	r.mapResponseToModel(ctx, updated, &plan, &resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Info(ctx, "Updated sink consumer resource", map[string]any{"id": consumerID})
}

// Delete deletes a sink consumer resource
func (r *SinkConsumerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SinkConsumerResourceModel

	// Read Terraform current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete
	consumerID := data.ID.ValueString()
	err := r.client.DeleteSinkConsumer(ctx, consumerID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Sink Consumer",
			"Could not delete sink consumer ID "+consumerID+": "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Deleted sink consumer resource", map[string]any{"id": consumerID})
	// State is automatically removed by Terraform after successful Delete
}

// ImportState imports an existing sink consumer resource by ID
func (r *SinkConsumerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID: terraform import sequin_sink_consumer.example <consumer-id>
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToModel maps API response to Terraform model
func (r *SinkConsumerResource) mapResponseToModel(ctx context.Context, response *client.SinkConsumerResponse, model *SinkConsumerResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(response.ID)
	model.Name = types.StringValue(response.Name)
	model.Status = types.StringValue(response.Status)
	model.Database = types.StringValue(response.Database)

	// Map source — treat empty source (no filters) as null to avoid drift
	sourceAttrTypes := map[string]attr.Type{
		"include_schemas": types.ListType{ElemType: types.StringType},
		"exclude_schemas": types.ListType{ElemType: types.StringType},
		"include_tables":  types.ListType{ElemType: types.StringType},
		"exclude_tables":  types.ListType{ElemType: types.StringType},
	}
	sourceHasData := response.Source != nil &&
		(len(response.Source.IncludeSchemas) > 0 ||
			len(response.Source.ExcludeSchemas) > 0 ||
			len(response.Source.IncludeTables) > 0 ||
			len(response.Source.ExcludeTables) > 0)

	if sourceHasData {
		sourceAttrs := map[string]attr.Value{
			"include_schemas": types.ListNull(types.StringType),
			"exclude_schemas": types.ListNull(types.StringType),
			"include_tables":  types.ListNull(types.StringType),
			"exclude_tables":  types.ListNull(types.StringType),
		}

		if len(response.Source.IncludeSchemas) > 0 {
			list, d := types.ListValueFrom(ctx, types.StringType, response.Source.IncludeSchemas)
			diags.Append(d...)
			sourceAttrs["include_schemas"] = list
		}
		if len(response.Source.ExcludeSchemas) > 0 {
			list, d := types.ListValueFrom(ctx, types.StringType, response.Source.ExcludeSchemas)
			diags.Append(d...)
			sourceAttrs["exclude_schemas"] = list
		}
		if len(response.Source.IncludeTables) > 0 {
			list, d := types.ListValueFrom(ctx, types.StringType, response.Source.IncludeTables)
			diags.Append(d...)
			sourceAttrs["include_tables"] = list
		}
		if len(response.Source.ExcludeTables) > 0 {
			list, d := types.ListValueFrom(ctx, types.StringType, response.Source.ExcludeTables)
			diags.Append(d...)
			sourceAttrs["exclude_tables"] = list
		}

		obj, d := types.ObjectValue(sourceAttrTypes, sourceAttrs)
		diags.Append(d...)
		model.Source = obj
	} else {
		model.Source = types.ObjectNull(sourceAttrTypes)
	}

	// Map tables
	tablesList := make([]attr.Value, len(response.Tables))
	for i, table := range response.Tables {
		tableAttrs := map[string]attr.Value{
			"name": types.StringValue(table.Name),
		}

		if len(table.GroupColumnNames) > 0 {
			list, d := types.ListValueFrom(ctx, types.StringType, table.GroupColumnNames)
			diags.Append(d...)
			tableAttrs["group_column_names"] = list
		} else {
			tableAttrs["group_column_names"] = types.ListNull(types.StringType)
		}

		obj, d := types.ObjectValue(map[string]attr.Type{
			"name":                types.StringType,
			"group_column_names": types.ListType{ElemType: types.StringType},
		}, tableAttrs)
		diags.Append(d...)
		tablesList[i] = obj
	}
	list, d := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":                types.StringType,
			"group_column_names": types.ListType{ElemType: types.StringType},
		},
	}, tablesList)
	diags.Append(d...)
	model.Tables = list

	// Map actions
	if len(response.Actions) > 0 {
		list, d := types.ListValueFrom(ctx, types.StringType, response.Actions)
		diags.Append(d...)
		model.Actions = list
	} else {
		model.Actions = types.ListNull(types.StringType)
	}

	// Map destination
	destAttrs := map[string]attr.Value{
		"type":                  types.StringValue(response.Destination.Type),
		"hosts":                 types.StringNull(),
		"topic":                 types.StringNull(),
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

	// Populate non-empty fields
	if response.Destination.Hosts != "" {
		destAttrs["hosts"] = types.StringValue(response.Destination.Hosts)
	}
	if response.Destination.Topic != "" {
		destAttrs["topic"] = types.StringValue(response.Destination.Topic)
	}
	if response.Destination.TLS != nil {
		destAttrs["tls"] = types.BoolValue(*response.Destination.TLS)
	}
	if response.Destination.Username != "" {
		destAttrs["username"] = types.StringValue(response.Destination.Username)
	}
	// Preserve values from existing state that the API doesn't return
	if !model.Destination.IsNull() {
		origDestAttrs := model.Destination.Attributes()
		// Preserve sensitive fields (API doesn't return them)
		if origPassword, ok := origDestAttrs["password"].(types.String); ok && !origPassword.IsNull() {
			destAttrs["password"] = origPassword
		}
		if origAWSAccessKey, ok := origDestAttrs["aws_access_key_id"].(types.String); ok && !origAWSAccessKey.IsNull() {
			destAttrs["aws_access_key_id"] = origAWSAccessKey
		}
		if origAWSSecretKey, ok := origDestAttrs["aws_secret_access_key"].(types.String); ok && !origAWSSecretKey.IsNull() {
			destAttrs["aws_secret_access_key"] = origAWSSecretKey
		}
		if origSecretKey, ok := origDestAttrs["secret_access_key"].(types.String); ok && !origSecretKey.IsNull() {
			destAttrs["secret_access_key"] = origSecretKey
		}
		if origAccessKey, ok := origDestAttrs["access_key_id"].(types.String); ok && !origAccessKey.IsNull() {
			destAttrs["access_key_id"] = origAccessKey
		}
		// Preserve topic from state if API returns empty (e.g. when routing overrides topic)
		if response.Destination.Topic == "" {
			if origTopic, ok := origDestAttrs["topic"].(types.String); ok && !origTopic.IsNull() {
				destAttrs["topic"] = origTopic
			}
		}
	}
	if response.Destination.SASLMechanism != "" {
		destAttrs["sasl_mechanism"] = types.StringValue(response.Destination.SASLMechanism)
	}
	if response.Destination.AWSRegion != "" {
		destAttrs["aws_region"] = types.StringValue(response.Destination.AWSRegion)
	}
	if response.Destination.QueueURL != "" {
		destAttrs["queue_url"] = types.StringValue(response.Destination.QueueURL)
	}
	if response.Destination.Region != "" {
		destAttrs["region"] = types.StringValue(response.Destination.Region)
	}
	if response.Destination.AccessKeyID != "" {
		destAttrs["access_key_id"] = types.StringValue(response.Destination.AccessKeyID)
	}
	if response.Destination.SecretAccessKey != "" {
		destAttrs["secret_access_key"] = types.StringValue(response.Destination.SecretAccessKey)
	}
	if response.Destination.IsFIFO != nil {
		destAttrs["is_fifo"] = types.BoolValue(*response.Destination.IsFIFO)
	}
	if response.Destination.StreamARN != "" {
		destAttrs["stream_arn"] = types.StringValue(response.Destination.StreamARN)
	}
	if response.Destination.HTTPEndpoint != "" {
		destAttrs["http_endpoint"] = types.StringValue(response.Destination.HTTPEndpoint)
	}
	if response.Destination.HTTPEndpointPath != "" {
		destAttrs["http_endpoint_path"] = types.StringValue(response.Destination.HTTPEndpointPath)
	}
	if response.Destination.Batch != nil {
		destAttrs["batch"] = types.BoolValue(*response.Destination.Batch)
	}

	destObj, d := types.ObjectValue(map[string]attr.Type{
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
	}, destAttrs)
	diags.Append(d...)
	model.Destination = destObj

	// Optional string fields — API returns "none" for unset values, treat as null
	if response.Filter != "" && response.Filter != "none" {
		model.Filter = types.StringValue(response.Filter)
	} else if model.Filter.IsNull() || model.Filter.IsUnknown() {
		model.Filter = types.StringNull()
	}
	if response.Transform != "" && response.Transform != "none" {
		model.Transform = types.StringValue(response.Transform)
	} else {
		model.Transform = types.StringNull()
	}
	if response.Enrichment != "" && response.Enrichment != "none" {
		model.Enrichment = types.StringValue(response.Enrichment)
	} else {
		model.Enrichment = types.StringNull()
	}
	if response.Routing != "" && response.Routing != "none" {
		model.Routing = types.StringValue(response.Routing)
	} else if model.Routing.IsNull() || model.Routing.IsUnknown() {
		model.Routing = types.StringNull()
	}
	model.MessageGrouping = types.BoolValue(response.MessageGrouping)
	model.BatchSize = types.Int64Value(int64(response.BatchSize))
	if response.MaxRetryCount != nil {
		model.MaxRetryCount = types.Int64Value(int64(*response.MaxRetryCount))
	} else {
		model.MaxRetryCount = types.Int64Null()
	}
	model.LoadSheddingPolicy = types.StringValue(response.LoadSheddingPolicy)
	model.TimestampFormat = types.StringValue(response.TimestampFormat)

	// Status info — only overwrite if API returned actual data
	statusInfoAttrTypes := map[string]attr.Type{
		"state":      types.StringType,
		"created_at": types.StringType,
		"updated_at": types.StringType,
		"last_error": types.StringType,
	}
	statusInfoHasData := response.StatusInfo.State != "" ||
		response.StatusInfo.CreatedAt != "" ||
		response.StatusInfo.UpdatedAt != ""

	if statusInfoHasData {
		statusInfoAttrs := map[string]attr.Value{
			"state":      types.StringValue(response.StatusInfo.State),
			"created_at": types.StringValue(response.StatusInfo.CreatedAt),
			"updated_at": types.StringValue(response.StatusInfo.UpdatedAt),
			"last_error": types.StringValue(response.StatusInfo.LastError),
		}
		statusInfoObj, d := types.ObjectValue(statusInfoAttrTypes, statusInfoAttrs)
		diags.Append(d...)
		model.StatusInfo = statusInfoObj
	} else if model.StatusInfo.IsNull() || model.StatusInfo.IsUnknown() {
		// API didn't return status_info and we have no prior state — set empty (must be known after apply)
		emptyAttrs := map[string]attr.Value{
			"state":      types.StringValue(""),
			"created_at": types.StringValue(""),
			"updated_at": types.StringValue(""),
			"last_error": types.StringValue(""),
		}
		statusInfoObj, d := types.ObjectValue(statusInfoAttrTypes, emptyAttrs)
		diags.Append(d...)
		model.StatusInfo = statusInfoObj
	}
	// else: keep existing state value (don't overwrite with empty data)
}
