package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1connect "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1/apiv1connect"
)

type ClientSet struct {
	TeamId                string
	ClusterMutationClient apiv1connect.ClusterMutationServiceClient
	K8SServiceClient      apiv1connect.K8SServiceClient
	RecommendationClient  apiv1connect.K8SRecommendationServiceClient
}

// Ensure DevzeroProvider satisfies various provider interfaces.
var _ provider.Provider = &DevzeroProvider{}

// DevzeroProvider defines the provider implementation.
type DevzeroProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version    string
	defaultURL string
}

// DevzeroProviderModel describes the provider data model.
type DevzeroProviderModel struct {
	URL    types.String `tfsdk:"url"`
	TeamId types.String `tfsdk:"team_id"`
	Token  types.String `tfsdk:"token"`
}

func (p *DevzeroProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "devzero"
	resp.Version = p.version
}

func (p *DevzeroProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Devzero API URL",
				Optional:            true,
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Devzero Team ID. You can retrieve it from your [Devzero Organization Settings](https://www.devzero.io/organization-settings/account)",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token used to authenticate with the Devzero API. For more information, see the [Devzero documentation](https://www.devzero.io/docs/platform/admin/personal-access-tokens).",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *DevzeroProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data DevzeroProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that the provider is not being initialized with unknown values (i.e: when initializing the provider with output values)

	if data.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown Devzero API URL",
			"The provider cannot create the Devzero API client as there is an unknown configuration value for the Devzero API URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the DEVZERO_URL environment variable.",
		)
	}

	if data.TeamId.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("team_id"),
			"Unknown Devzero Team ID",
			"The provider cannot create the Devzero API client as there is an unknown configuration value for the Devzero Team ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the DEVZERO_TEAM_ID environment variable.",
		)
	}

	if data.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Devzero API Token",
			"The provider cannot create the Devzero API client as there is an unknown configuration value for the Devzero API Token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the DEVZERO_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default to environment variables, but override with Terraform configuration

	url := os.Getenv("DEVZERO_URL")
	teamId := os.Getenv("DEVZERO_TEAM_ID")
	token := os.Getenv("DEVZERO_TOKEN")

	if !data.URL.IsNull() {
		url = data.URL.ValueString()
	}

	if !data.TeamId.IsNull() {
		teamId = data.TeamId.ValueString()
	}

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	// If any of the expected configurations are missing, then return errors or set the defaults values

	if url == "" {
		url = p.defaultURL
	}

	if teamId == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("team_id"),
			"Missing Devzero Team ID",
			"The provider cannot create the Devzero API client as there is no Devzero Team ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the DEVZERO_TEAM_ID environment variable.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Devzero API Token",
			"The provider cannot create the Devzero API client as there is no Devzero API Token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the DEVZERO_TOKEN environment variable."+
				"If either is already set, ensure the value is not empty."+
				"For more information, see the [Devzero documentation](https://www.devzero.io/docs/platform/admin/personal-access-tokens).",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := http.DefaultClient

	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
			return next(ctx, req)
		}
	})

	// Create the Devzero API client
	clientset := &ClientSet{
		TeamId: teamId,
		ClusterMutationClient: apiv1connect.NewClusterMutationServiceClient(
			client,
			url,
			connect.WithGRPC(),
			connect.WithInterceptors(authInterceptor),
		),
		K8SServiceClient: apiv1connect.NewK8SServiceClient(
			client,
			url,
			connect.WithGRPC(),
			connect.WithInterceptors(authInterceptor),
		),
		RecommendationClient: apiv1connect.NewK8SRecommendationServiceClient(
			client,
			url,
			connect.WithGRPC(),
			connect.WithInterceptors(authInterceptor),
		),
	}

	// Example client configuration for data sources and resources
	resp.DataSourceData = clientset
	resp.ResourceData = clientset
}

func (p *DevzeroProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
		NewWorkloadPolicyResource,
		NewWorkloadPolicyTargetResource,
	}
}

func (p *DevzeroProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DevzeroProvider{
			version:    version,
			defaultURL: "https://dakr.devzero.io",
		}
	}
}
