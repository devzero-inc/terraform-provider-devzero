package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NodePolicyTargetResource{}
var _ resource.ResourceWithConfigure = &NodePolicyTargetResource{}
var _ resource.ResourceWithImportState = &NodePolicyTargetResource{}

func NewNodePolicyTargetResource() resource.Resource {
	return &NodePolicyTargetResource{}
}

// NodePolicyTargetResource defines the resource implementation.
type NodePolicyTargetResource struct {
	client *ClientSet
}

// NodePolicyTargetResourceModel describes the resource data model.
type NodePolicyTargetResourceModel struct {
	Id          types.String `tfsdk:"id"`
	PolicyId    types.String `tfsdk:"policy_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	ClusterIds  types.List   `tfsdk:"cluster_ids"`
}

func (r *NodePolicyTargetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_policy_target"
}

func (r *NodePolicyTargetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a node policy to specific clusters. Node policy targets determine which clusters a node policy applies to.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Unique identifier of the node policy target",
				MarkdownDescription: "Unique identifier of the node policy target. Managed by the provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_id": schema.StringAttribute{
				Description:         "Node policy to attach this target to",
				MarkdownDescription: "Node policy to attach this target to. Must reference an existing `devzero_node_policy` resource ID.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Human-friendly name for the target",
				MarkdownDescription: "Human-friendly name for the target. Used for display in the DevZero UI.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Free-form description of the target",
				MarkdownDescription: "Free-form description of the target to help others understand its purpose.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers:       []planmodifier.String{preserveStringStateOverDefault()},
			},
			"enabled": schema.BoolAttribute{
				Description:         "Whether this target is active",
				MarkdownDescription: "Whether this target is active. When false, the node policy will not be applied to the specified clusters.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers:       []planmodifier.Bool{preserveBoolStateOverDefault()},
			},
			"cluster_ids": schema.ListAttribute{
				Description:         "Cluster ID to apply the node policy to (at most one)",
				MarkdownDescription: "Cluster ID to apply the node policy to. The API accepts at most one cluster per target; create one target per cluster.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
			},
		},
	}
}

func (r *NodePolicyTargetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NodePolicyTargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NodePolicyTargetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target := data.toProto(ctx, &resp.Diagnostics, r.client.TeamId)
	if resp.Diagnostics.HasError() {
		return
	}

	createNodePolicyTargetsReq := &apiv1.CreateNodePolicyTargetsRequest{
		Targets: []*apiv1.NodePolicyTarget{target}, // Wrap single target in array
	}

	createNodePolicyTargetsResp, err := r.client.RecommendationClient.CreateNodePolicyTargets(ctx, connect.NewRequest(createNodePolicyTargetsReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create node policy target, got error: %s", err))
		return
	}
	if len(createNodePolicyTargetsResp.Msg.Targets) == 0 {
		resp.Diagnostics.AddError("Client Error", "Node policy target not created")
		return
	}

	data.fromProto(createNodePolicyTargetsResp.Msg.Targets[0])

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a node policy target resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePolicyTargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NodePolicyTargetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// List all targets and find the one with matching ID
	listNodePolicyTargetsReq := &apiv1.ListNodePolicyTargetsRequest{
		TeamId: r.client.TeamId,
	}

	listNodePolicyTargetsResp, err := r.client.RecommendationClient.ListNodePolicyTargets(ctx, connect.NewRequest(listNodePolicyTargetsReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list node policy targets, got error: %s", err))
		return
	}

	// Find the target with matching ID
	var foundTarget *apiv1.NodePolicyTarget
	for _, target := range listNodePolicyTargetsResp.Msg.Targets {
		if target.TargetId == data.Id.ValueString() {
			foundTarget = target
			break
		}
	}

	if foundTarget == nil {
		// Deleted out of band — drop it from state so the plan recreates it.
		resp.State.RemoveResource(ctx)
		return
	}

	data.fromProto(foundTarget)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePolicyTargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NodePolicyTargetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateNodePolicyTargetReq := &apiv1.UpdateNodePolicyTargetRequest{
		Target: data.toProto(ctx, &resp.Diagnostics, r.client.TeamId),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	updateNodePolicyTargetResp, err := r.client.RecommendationClient.UpdateNodePolicyTarget(ctx, connect.NewRequest(updateNodePolicyTargetReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update node policy target, got error: %s", err))
		return
	}

	if updateNodePolicyTargetResp.Msg.Target == nil {
		resp.Diagnostics.AddError("Client Error", "Node policy target not updated")
		return
	}

	data.fromProto(updateNodePolicyTargetResp.Msg.Target)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePolicyTargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NodePolicyTargetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The API has no DeleteNodePolicyTarget RPC. Leaving the target enabled
	// would keep applying the policy after destroy, so disable it instead.
	target := data.toProto(ctx, &resp.Diagnostics, r.client.TeamId)
	if resp.Diagnostics.HasError() {
		return
	}
	target.Enabled = false

	_, err := r.client.RecommendationClient.UpdateNodePolicyTarget(ctx, connect.NewRequest(&apiv1.UpdateNodePolicyTargetRequest{
		Target: target,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable node policy target on destroy, got error: %s", err))
		return
	}

	tflog.Warn(ctx, "The API has no delete endpoint for node policy targets; the target was disabled and removed from state, but the row remains in the backend.")
}

func (r *NodePolicyTargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// toProto converts Terraform model to protobuf message.
func (m *NodePolicyTargetResourceModel) toProto(ctx context.Context, diags *diag.Diagnostics, teamId string) *apiv1.NodePolicyTarget {
	clusterIds, err := getStringList(ctx, m.ClusterIds.Elements())
	if err != nil {
		diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert cluster IDs: %s", err))
		return nil
	}

	return &apiv1.NodePolicyTarget{
		TargetId:    m.Id.ValueString(),
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		TeamId:      teamId,
		ClusterIds:  clusterIds,
		PolicyId:    m.PolicyId.ValueString(),
		Enabled:     m.Enabled.ValueBool(),
	}
}

// fromProto converts protobuf message to Terraform model.
func (m *NodePolicyTargetResourceModel) fromProto(target *apiv1.NodePolicyTarget) {
	m.Id = types.StringValue(target.TargetId)
	m.PolicyId = types.StringValue(target.PolicyId)
	m.Name = types.StringValue(target.Name)
	m.Description = types.StringValue(target.Description)
	m.Enabled = types.BoolValue(target.Enabled)
	m.ClusterIds = types.ListValueMust(types.StringType, fromStringList(target.ClusterIds))
}
