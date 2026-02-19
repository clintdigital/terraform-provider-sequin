package provider

import (
	"context"
	"os"

	"github.com/clintdigital/terraform-provider-sequin/internal/client"
	"github.com/clintdigital/terraform-provider-sequin/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the provider.Provider interface
var _ provider.Provider = &SequinProvider{}

// SequinProvider defines the provider implementation.
type SequinProvider struct {
	version string
}

// SequinProviderModel describes the provider data model.
type SequinProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
}

// New creates a new provider instance
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SequinProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *SequinProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sequin"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *SequinProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Sequin Stream API resources including databases, sink consumers, and backfills.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Sequin API endpoint URL. Can also be set via SEQUIN_ENDPOINT environment variable.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "Sequin API authentication key. Can also be set via SEQUIN_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares the provider client for data sources and resources.
func (p *SequinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Sequin provider")

	var config SequinProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Allow environment variables to override config
	endpoint := os.Getenv("SEQUIN_ENDPOINT")
	apiKey := os.Getenv("SEQUIN_API_KEY")

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	// Validate required configuration
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Sequin API Endpoint",
			"The provider cannot create the Sequin API client as there is a missing or empty value for the endpoint. "+
				"Set the endpoint value in the configuration or use the SEQUIN_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Sequin API Key",
			"The provider cannot create the Sequin API client as there is a missing or empty value for the API key. "+
				"Set the api_key value in the configuration or use the SEQUIN_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API client
	c := client.New(endpoint, apiKey, p.version)

	// Make the client available to resources and data sources
	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured Sequin provider", map[string]any{"endpoint": endpoint})
}

// Resources defines the resources implemented in the provider.
func (p *SequinProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatabaseResource,
		resources.NewSinkConsumerResource,
		resources.NewBackfillResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *SequinProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Data sources can be added here if needed
	}
}
