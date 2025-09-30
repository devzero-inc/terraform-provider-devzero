package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Token types.String `tfsdk:"token"`
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
	// If the resource is being created, skip forcing a rotation during plan
	if req.State.Raw.IsNull() {
		return
	}

	var data ClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the prior token is empty, mark the planned token as unknown so that
	// Terraform plans an apply which will rotate the token during Update.
	if data.Token.IsNull() || data.Token.ValueString() == "" {
		// Only attempt to set if plan is available
		if !req.Plan.Raw.IsNull() {
			// Set token to unknown in plan
			err := resp.Plan.SetAttribute(ctx, path.Root("token"), types.StringUnknown())
			if err != nil {
				resp.Diagnostics.AddError("Plan Error", fmt.Sprintf("Unable to mark token unknown in plan: %s", err))
				return
			}
		}
	}
}

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createClusterReq := &apiv1.CreateClusterRequest{
		TeamId:      r.client.TeamId,
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

	getClusterReq := &apiv1.GetClusterRequest{
		TeamId:    r.client.TeamId,
		ClusterId: data.Id.ValueString(),
	}

	getClusterResp, err := r.client.K8SServiceClient.GetCluster(ctx, connect.NewRequest(getClusterReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get cluster, got error: %s", err))
		return
	}

	if getClusterResp.Msg.Cluster == nil {
		resp.Diagnostics.AddError("Client Error", "Cluster not found")
		return
	}

	name := getClusterResp.Msg.Cluster.CustomName
	if name == "" {
		name = getClusterResp.Msg.Cluster.Name
	}

	data.Name = types.StringValue(name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateClusterReq := &apiv1.UpdateClusterRequest{
		TeamId:      r.client.TeamId,
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

	data.Name = types.StringValue(updateClusterResp.Msg.Cluster.CustomName)

	// If prior token was empty, rotate it now and persist the new token in state
	if data.Token.IsNull() || data.Token.IsUnknown() || data.Token.ValueString() == "" {
		resetReq := &apiv1.ResetClusterTokenRequest{
			TeamId:    r.client.TeamId,
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
		TeamId:    r.client.TeamId,
		ClusterId: data.Id.ValueString(),
	}

	_, err := r.client.ClusterMutationClient.DeleteCluster(ctx, connect.NewRequest(deleteClusterReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error: %s", err))
		return
	}
}

func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
