package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &hibobProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &hibobProvider{
			version: version,
		}
	}
}

// hibobProvider is the provider implementation.
type hibobProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// hibobProviderModel maps provider schema data to a Go type.
type hibobProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// Metadata returns the provider type name.
func (p *hibobProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hibob"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *hibobProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with hibob.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "Username for hibob API.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for hibob API.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *hibobProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring hibob client")

	// Retrieve provider data from configuration
	var config hibobProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown hibob API Username",
			"The provider cannot create the hibob API client as there is an unknown configuration value for the hibob API username. "+
				"Either target apply the source of the value first or set the value statically in the configuration.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown hibob API Password",
			"The provider cannot create the hibob API client as there is an unknown configuration value for the hibob API password. "+
				"Either target apply the source of the value first or set the value statically in the configuration.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	username := config.Username.ValueString()
	password := config.Password.ValueString()

	ctx = tflog.SetField(ctx, "hibob_username", username)
	ctx = tflog.SetField(ctx, "hibob_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "hibob_password")

	tflog.Debug(ctx, "Creating hibob client")

	// Create a new hibob client using the configuration values
	//client, err := hibob.NewClient(&host, &username, &password)
	request, _ := http.NewRequest("GET", "https://api.hibob.com/v1/company/people/fields", nil)

	request.Header.Add("accept", "application/json")
	request.SetBasicAuth(username, password)

	authorizationHeader := request.Header.Get("Authorization")

	res, err := http.DefaultClient.Do(request)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create hibob API Client",
			"An unexpected error occurred when creating the hibob API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"hibob Client Error: "+err.Error(),
		)
		return
	}

	defer res.Body.Close()
	_, err = io.ReadAll(res.Body)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create hibob API Client",
			"An unexpected error occurred when creating the hibob API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"hibob Client Error: "+err.Error(),
		)
		return
	}

	// Make the hibob client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = authorizationHeader
	//resp.ResourceData = client

	tflog.Info(ctx, "Configured hibob client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *hibobProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEmployeesDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *hibobProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
