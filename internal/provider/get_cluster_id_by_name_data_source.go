package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ClusterIDByNameDataSource{}
var _ datasource.DataSourceWithConfigure = &ClusterIDByNameDataSource{}

func NewClusterIDByNameDataSource() datasource.DataSource {
	return &ClusterIDByNameDataSource{}
}

type ClusterIDByNameDataSource struct {
	client *ClientSet
}

type ClusterIDByNameDataSourceModel struct {
	TeamID        types.String `tfsdk:"team_id"`
	Name          types.String `tfsdk:"name"`
	Region        types.String `tfsdk:"region"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Liveness      types.String `tfsdk:"liveness"`
	ClusterID     types.String `tfsdk:"cluster_id"`
}

func (d *ClusterIDByNameDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_get_cluster_id_by_name"
}

func (d *ClusterIDByNameDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing cluster by team ID and name, returning its ID.",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "The team ID to search within. Defaults to the provider team_id if not set.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The cluster name to look up.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Optional region filter, e.g. \"us-east-1\".",
				Optional:            true,
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Optional cloud provider filter. One of: 'AWS', 'GCP', 'AKS', 'OCI'.",
				Optional:            true,
			},
			"liveness": schema.StringAttribute{
				MarkdownDescription: "Controls liveness filtering: IGNORE, PREFER_LIVE, or REQUIRE_LIVE.",
				Optional:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster matching the given team and name.",
				Computed:            true,
			},
		},
	}
}

func (d *ClusterIDByNameDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientSet)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ClientSet, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ClusterIDByNameDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClusterIDByNameDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := d.client.TeamId
	if !data.TeamID.IsNull() && !data.TeamID.IsUnknown() && data.TeamID.ValueString() != "" {
		teamID = data.TeamID.ValueString()
	}

	rpcReq := &apiv1.GetClusterIDByNameRequest{
		TeamId: teamID,
		Name:   data.Name.ValueString(),
	}

	if !data.Region.IsNull() && !data.Region.IsUnknown() {
		v := data.Region.ValueString()
		rpcReq.Region = &v
	}

	if !data.CloudProvider.IsNull() && !data.CloudProvider.IsUnknown() {
		v := data.CloudProvider.ValueString()
		rpcReq.CloudProvider = &v
	}

	if !data.Liveness.IsNull() && !data.Liveness.IsUnknown() {
		livenessStr := "CLUSTER_LIVENESS_PREFERENCE_" + data.Liveness.ValueString()
		val, ok := apiv1.ClusterLivenessPreference_value[livenessStr]
		if !ok {
			resp.Diagnostics.AddError(
				"Invalid Liveness Value",
				fmt.Sprintf("Invalid liveness value %q, must be one of: IGNORE, PREFER_LIVE, REQUIRE_LIVE.", data.Liveness.ValueString()),
			)
			return
		}
		liveness := apiv1.ClusterLivenessPreference(val)
		rpcReq.Liveness = &liveness
	}

	rpcResp, err := d.client.ClusterServiceClient.GetClusterIDByName(ctx, connect.NewRequest(rpcReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get cluster ID by name, got error: %s", err))
		return
	}

	if rpcResp.Msg.Id == "" {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("No cluster found with name %q in team %q.", data.Name.ValueString(), teamID))
		return
	}

	data.TeamID = types.StringValue(teamID)
	data.ClusterID = types.StringValue(rpcResp.Msg.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}