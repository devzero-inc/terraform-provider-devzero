package provider

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithConfigure = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}
var _ resource.ResourceWithModifyPlan = &ClusterResource{}

func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

// ExampleResource defines the resource implementation.
type ClusterResource struct {
	client *ClientSet
}

// ExampleResourceModel describes the resource data model.
type ClusterResourceModel struct {
	Id     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Token  types.String `tfsdk:"token"`
	TeamId types.String `tfsdk:"team_id"`
}

func (r *ClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *ClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Cluster resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the cluster",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the cluster",
				Required:    true,
			},
			"token": schema.StringAttribute{
				Description: "Token of the cluster",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "ID of the team the cluster is registered in, such as a `devzero_team`'s `id`. " +
					"Defaults to the provider's `team_id`. It is set when the cluster is registered: " +
					"on an existing cluster, a plan that changes a known `team_id`, or leaves it unknown until apply, is refused.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *ClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientSet)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ClientSet, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ClusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to check on a create or a destroy.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var data ClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	var planned ClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An existing cluster refuses an unknown team_id: it resolves at apply,
	// after what produces it has run.
	if !data.TeamId.IsNull() && !data.TeamId.IsUnknown() {
		if planned.TeamId.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("team_id"),
				"A cluster's team cannot change",
				fmt.Sprintf("Cluster %s is registered in team %s and stays there, but this plan leaves its team_id unknown until apply, "+
					"as when the devzero_team it names is about to be created. "+
					"A devzero_team whose address was renamed or replaced needs a moved block from its old address, so Terraform keeps the team rather than starting a new one. "+
					"Otherwise set team_id to %q, or remove team_id to keep the cluster's team.",
					data.Id.ValueString(), data.TeamId.ValueString(), data.TeamId.ValueString()),
			)
			return
		}
		if !planned.TeamId.Equal(data.TeamId) {
			resp.Diagnostics.AddAttributeError(
				path.Root("team_id"),
				"A cluster's team cannot change",
				fmt.Sprintf("Cluster %s is registered in team %s. Terraform never moves a cluster to another team or re-creates it there. "+
					"Set team_id back to %q, or remove team_id to keep the cluster's team. To register a cluster in team %s, add a new devzero_cluster.",
					data.Id.ValueString(), data.TeamId.ValueString(), data.TeamId.ValueString(), planned.TeamId.ValueString()),
			)
			return
		}
	}

	// If the prior token is empty, mark the planned token as unknown so that
	// Terraform plans an apply which will rotate the token during Update.
	if data.Token.IsNull() || data.Token.ValueString() == "" {
		err := resp.Plan.SetAttribute(ctx, path.Root("token"), types.StringUnknown())
		if err != nil {
			resp.Diagnostics.AddError("Plan Error", fmt.Sprintf("Unable to mark token unknown in plan: %s", err))
			return
		}
	}
}

// teamOf is the team every call for a cluster goes to: its own team_id, or
// the provider's while that is null or unknown.
func (r *ClusterResource) teamOf(teamID types.String) string {
	if teamID.IsNull() || teamID.IsUnknown() || teamID.ValueString() == "" {
		return r.client.TeamId
	}
	return teamID.ValueString()
}

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamID := r.teamOf(data.TeamId)
	createClusterReq := &apiv1.CreateClusterRequest{
		TeamId:      teamID,
		ClusterName: data.Name.ValueString(),
	}

	createClusterResp, err := r.client.ClusterMutationClient.CreateCluster(ctx, connect.NewRequest(createClusterReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error: %s", err))
		return
	}
	if createClusterResp.Msg.Cluster == nil || createClusterResp.Msg.Token == "" {
		resp.Diagnostics.AddError("Client Error", "Cluster not created")
		return
	}

	// Set the state
	data.Id = types.StringValue(createClusterResp.Msg.Cluster.Id)
	data.Token = types.StringValue(createClusterResp.Msg.Token)
	data.TeamId = types.StringValue(teamID)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamInferred := data.TeamId.ValueString() == ""
	teamID := r.teamOf(data.TeamId)
	getClusterReq := &apiv1.GetClusterRequest{
		TeamId:    teamID,
		ClusterId: data.Id.ValueString(),
	}

	getClusterResp, err := r.client.K8SServiceClient.GetCluster(ctx, connect.NewRequest(getClusterReq))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			r.clusterGone(ctx, resp, data.Id.ValueString(), teamID, teamInferred)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get cluster, got error: %s", err))
		return
	}

	if getClusterResp.Msg.Cluster == nil {
		r.clusterGone(ctx, resp, data.Id.ValueString(), teamID, teamInferred)
		return
	}

	name := getClusterResp.Msg.Cluster.CustomName
	if name == "" {
		name = getClusterResp.Msg.Cluster.Name
	}

	data.Name = types.StringValue(name)
	data.TeamId = types.StringValue(teamID)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// clusterGone drops a missing cluster from state only when its stored team is
// the provider's. One whose team was inferred from the provider's may be
// registered in the team the provider had before, so it fails, as does any
// other team's.
func (r *ClusterResource) clusterGone(ctx context.Context, resp *resource.ReadResponse, clusterID, teamID string, teamInferred bool) {
	if teamInferred {
		resp.Diagnostics.AddError(
			"Cluster not found in the provider's team",
			fmt.Sprintf("Cluster %s is not in team %s, the provider's team. Its state has no team_id, like every state written by a provider release before team_id, "+
				"so the team it is registered in is not known, and Terraform keeps it in state rather than registering a new cluster in its place. "+
				"If it is registered in another team, set the provider's team_id back to that team for one terraform apply -refresh-only, which stores the team in the cluster's state, "+
				"or remove it from state and import it as <team_id>/<cluster_id>. "+
				"If the cluster is gone, remove it from state (terraform state rm, or a removed block). "+
				"Setting team_id on this devzero_cluster in configuration does not help: the refresh reads the cluster's team from state.",
				clusterID, teamID),
		)
		return
	}
	if teamID == r.client.TeamId {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.AddError(
		"Cluster not found in its team",
		fmt.Sprintf("Cluster %s is not in team %s, or this provider's token no longer reaches that team. "+
			"Terraform keeps it in state rather than registering a new cluster in its place. "+
			"If the cluster or its team is gone, remove this devzero_cluster, and the team's devzero_team if this configuration started the team, "+
			"from configuration as well as from state (terraform state rm), or replace them with removed blocks. "+
			"Removed from state but kept in configuration, they start a new team and register the customer's cluster into it on the next apply.",
			clusterID, teamID),
	)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamID := r.teamOf(data.TeamId)
	updateClusterReq := &apiv1.UpdateClusterRequest{
		TeamId:      teamID,
		ClusterId:   data.Id.ValueString(),
		ClusterName: data.Name.ValueString(),
	}

	updateClusterResp, err := r.client.ClusterMutationClient.UpdateCluster(ctx, connect.NewRequest(updateClusterReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster, got error: %s", err))
		return
	}

	if updateClusterResp.Msg.Cluster == nil {
		resp.Diagnostics.AddError("Client Error", "Cluster not updated")
		return
	}

	updatedName := updateClusterResp.Msg.Cluster.CustomName
	if updatedName == "" {
		updatedName = updateClusterResp.Msg.Cluster.Name
	}
	data.Name = types.StringValue(updatedName)
	data.TeamId = types.StringValue(teamID)

	// If prior token was empty, rotate it now and persist the new token in state
	if data.Token.IsNull() || data.Token.IsUnknown() || data.Token.ValueString() == "" {
		resetReq := &apiv1.ResetClusterTokenRequest{
			TeamId:    teamID,
			ClusterId: data.Id.ValueString(),
		}
		resetResp, err := r.client.ClusterMutationClient.ResetClusterToken(ctx, connect.NewRequest(resetReq))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset cluster token, got error: %s", err))
			return
		}
		if resetResp.Msg.Token == "" {
			resp.Diagnostics.AddError("Client Error", "Cluster token reset returned empty token")
			return
		}
		data.Token = types.StringValue(resetResp.Msg.Token)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteClusterReq := &apiv1.DeleteClusterRequest{
		TeamId:    r.teamOf(data.TeamId),
		ClusterId: data.Id.ValueString(),
	}

	_, err := r.client.ClusterMutationClient.DeleteCluster(ctx, connect.NewRequest(deleteClusterReq))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error: %s", err))
		return
	}
}

// ImportState takes `<cluster_id>`, for a cluster of the provider's team, or
// `<team_id>/<cluster_id>`.
func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, clusterID, found := strings.Cut(req.ID, "/")
	if !found {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		return
	}
	if teamID == "" || clusterID == "" || strings.Contains(clusterID, "/") {
		resp.Diagnostics.AddError(
			"Unexpected import ID",
			fmt.Sprintf("Expected <cluster_id> or <team_id>/<cluster_id>, got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), clusterID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), teamID)...)
}
