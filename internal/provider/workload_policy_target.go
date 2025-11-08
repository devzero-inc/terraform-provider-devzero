// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkloadPolicyTargetResource{}
var _ resource.ResourceWithConfigure = &WorkloadPolicyTargetResource{}
var _ resource.ResourceWithImportState = &WorkloadPolicyTargetResource{}

func NewWorkloadPolicyTargetResource() resource.Resource {
	return &WorkloadPolicyTargetResource{}
}

// ExampleResource defines the resource implementation.
type WorkloadPolicyTargetResource struct {
	client *ClientSet
}

// ExampleResourceModel describes the resource data model.
type WorkloadPolicyTargetResourceModel struct {
	Id                types.String   `tfsdk:"id"`
	PolicyId          types.String   `tfsdk:"policy_id"`
	Name              types.String   `tfsdk:"name"`
	Description       types.String   `tfsdk:"description"`
	Priority          types.Int32    `tfsdk:"priority"`
	Enabled           types.Bool     `tfsdk:"enabled"`
	NamespaceSelector *LabelSelector `tfsdk:"namespace_selector"`
	WorkloadSelector  *LabelSelector `tfsdk:"workload_selector"`
	KindFilter        types.List     `tfsdk:"kind_filter"`
	NamePattern       *RegexPattern  `tfsdk:"name_pattern"`
	WorkloadNames     types.List     `tfsdk:"workload_names"`
	NodeGroupNames    types.List     `tfsdk:"node_group_names"`
	ClusterIds        types.List     `tfsdk:"cluster_ids"`
}

type LabelSelector struct {
	MatchLabels      types.Map  `tfsdk:"match_labels"`
	MatchExpressions types.List `tfsdk:"match_expressions"`
}

type MatchExpression struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.List   `tfsdk:"values"`
}

func (m MatchExpression) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":      types.StringType,
		"operator": types.StringType,
		"values":   types.ListType{ElemType: types.StringType},
	}
}

type RegexPattern struct {
	Pattern types.String `tfsdk:"pattern"`
	Flags   types.String `tfsdk:"flags"`
}

func (r *WorkloadPolicyTargetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_policy_target"
}

func (r *WorkloadPolicyTargetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	labelSelectorAttributes := map[string]schema.Attribute{
		"match_labels": schema.MapAttribute{
			Description:         "Exact label key/value pairs that the target must match",
			MarkdownDescription: "Exact label key/value pairs that the target must match. Keys and values must be strings. Example: `{ \"app\": \"api\", \"env\": \"prod\" }`.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"match_expressions": schema.ListNestedAttribute{
			Description:         "Advanced label selector requirements",
			MarkdownDescription: "Advanced label selector requirements. Each expression supports operators `In`, `NotIn`, `Exists`, `DoesNotExist`. Use `values` only with `In`/`NotIn`.",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"key": schema.StringAttribute{
						Description:         "Label key to evaluate",
						MarkdownDescription: "Label key to evaluate. Example: `app` or `kubernetes.io/name`.",
						Optional:            true,
					},
					"operator": schema.StringAttribute{
						Description:         "Label selection operator",
						MarkdownDescription: "Label selection operator. One of `In`, `NotIn`, `Exists`, `DoesNotExist`.",
						Optional:            true,
					},
					"values": schema.ListAttribute{
						Description:         "Values to compare against the key",
						MarkdownDescription: "Values to compare against the key. Required with `In`/`NotIn`; must be omitted with `Exists`/`DoesNotExist`.",
						Optional:            true,
						ElementType:         types.StringType,
					},
				},
			},
		},
	}

	regexPatternAttributes := map[string]schema.Attribute{
		"pattern": schema.StringAttribute{
			Description:         "Regular expression to match against the workload name",
			MarkdownDescription: "Regular expression applied to workload names. Uses RE2 syntax. Example: `^api-(staging|prod)-.*$`.",
			Optional:            true,
		},
		"flags": schema.StringAttribute{
			Description:         "Regex flags to modify matching behavior",
			MarkdownDescription: "Regex flags to modify matching behavior. Supported: `i` (case-insensitive), `m` (multi-line).",
			Optional:            true,
		},
	}

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Defines which workloads a policy applies to by selecting namespaces, workloads, names, and clusters. Combine selectors and filters to precisely target Kubernetes objects.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Unique identifier of the workload policy target",
				MarkdownDescription: "Unique identifier of the workload policy target. Managed by the provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_id": schema.StringAttribute{
				Description:         "Workload policy to attach this target to",
				MarkdownDescription: "Workload policy to attach this target to. Must reference an existing `devzero_workload_policy` resource ID.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Human-friendly name for this target",
				MarkdownDescription: "Human-friendly name for this target. Used for display in the DevZero UI.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Free-form description of the target",
				MarkdownDescription: "Free-form description of the target to help others understand its purpose.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"priority": schema.Int32Attribute{
				Description:         "Evaluation priority among multiple targets",
				MarkdownDescription: "Evaluation priority among multiple targets. Higher values take precedence when multiple targets overlap.",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(0),
			},
			"enabled": schema.BoolAttribute{
				Description:         "Enable or disable this target",
				MarkdownDescription: "Enable or disable this target. When disabled, the associated policy will not apply to the selected workloads.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"namespace_selector": schema.SingleNestedAttribute{
				Description:         "Select namespaces by labels",
				MarkdownDescription: "Select namespaces by labels. Uses the same semantics as Kubernetes label selectors.",
				Optional:            true,
				Attributes:          labelSelectorAttributes,
			},
			"workload_selector": schema.SingleNestedAttribute{
				Description:         "Select workloads by labels",
				MarkdownDescription: "Select workloads by labels. Applies to Kubernetes objects like Deployments, StatefulSets, DaemonSets, etc.",
				Optional:            true,
				Attributes:          labelSelectorAttributes,
			},
			"kind_filter": schema.ListAttribute{
				Description:         "Restrict matching to specific Kubernetes kinds",
				MarkdownDescription: "Restrict matching to specific Kubernetes kinds. Allowed values: `Pod`, `Job`, `Deployment`, `StatefulSet`, `DaemonSet`, `ReplicaSet`, `CronJob`, `ReplicationController`, `Rollout`.",
				Optional:            true,
				ElementType:         types.StringType,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"name_pattern": schema.SingleNestedAttribute{
				Description:         "Regex to match workload names",
				MarkdownDescription: "Regex to match workload names. Useful to target rollouts or name conventions (e.g., `^api-.*`).",
				Optional:            true,
				Attributes:          regexPatternAttributes,
			},
			"workload_names": schema.ListAttribute{
				Description:         "Explicit list of workload names to include",
				MarkdownDescription: "Explicit list of workload names to include",
				Optional:            true,
				ElementType:         types.StringType,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"node_group_names": schema.ListAttribute{
				Description:         "Restrict matching to specific node groups",
				MarkdownDescription: "Restrict matching to specific node groups by name",
				Optional:            true,
				ElementType:         types.StringType,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"cluster_ids": schema.ListAttribute{
				Description:         "Clusters where this target should apply",
				MarkdownDescription: "Clusters where this target should apply. Provide one or more cluster IDs from `devzero_cluster`.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *WorkloadPolicyTargetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkloadPolicyTargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkloadPolicyTargetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kindFilters, err := getKindFilters(ctx, data.KindFilter.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert kind filter to Terraform value, got error: %s", err))
		return
	}

	workloadNames, err := getStringList(ctx, data.WorkloadNames.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert workload name to Terraform value, got error: %s", err))
		return
	}

	nodeGroupNames, err := getStringList(ctx, data.NodeGroupNames.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert node group name to Terraform value, got error: %s", err))
		return
	}

	clusterIds, err := getStringList(ctx, data.ClusterIds.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert cluster ID to Terraform value, got error: %s", err))
		return
	}

	namespaceSelector, err := data.NamespaceSelector.toProto(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert namespace selector to Terraform value, got error: %s", err))
		return
	}
	workloadSelector, err := data.WorkloadSelector.toProto(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert workload selector to Terraform value, got error: %s", err))
		return
	}

	createWorkloadPolicyTargetReq := &apiv1.CreateWorkloadPolicyTargetRequest{
		TeamId:            r.client.TeamId,
		PolicyId:          data.PolicyId.ValueString(),
		Name:              data.Name.ValueString(),
		Description:       data.Description.ValueString(),
		Priority:          data.Priority.ValueInt32(),
		Enabled:           data.Enabled.ValueBool(),
		NamespaceSelector: namespaceSelector,
		WorkloadSelector:  workloadSelector,
		KindFilter:        kindFilters,
		NamePattern:       data.NamePattern.toProto(),
		WorkloadNames:     workloadNames,
		NodeGroupNames:    nodeGroupNames,
		ClusterIds:        clusterIds,
	}

	createWorkloadPolicyTargetResp, err := r.client.RecommendationClient.CreateWorkloadPolicyTarget(ctx, connect.NewRequest(createWorkloadPolicyTargetReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster, got error: %s", err))
		return
	}
	if createWorkloadPolicyTargetResp.Msg.Target == nil {
		resp.Diagnostics.AddError("Client Error", "Workload policy target not created")
		return
	}

	// Set the state
	data.fromProto(createWorkloadPolicyTargetResp.Msg.Target)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadPolicyTargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkloadPolicyTargetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getWorkloadPolicyTargetReq := &apiv1.GetWorkloadPolicyTargetRequest{
		TeamId:   r.client.TeamId,
		TargetId: data.Id.ValueString(),
	}

	getWorkloadPolicyTargetResp, err := r.client.RecommendationClient.GetWorkloadPolicyTarget(ctx, connect.NewRequest(getWorkloadPolicyTargetReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get cluster, got error: %s", err))
		return
	}

	if getWorkloadPolicyTargetResp.Msg.Target == nil {
		resp.Diagnostics.AddError("Client Error", "Workload policy target not found")
		return
	}

	data.fromProto(getWorkloadPolicyTargetResp.Msg.Target)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadPolicyTargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkloadPolicyTargetResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	kindFilters, err := getKindFilters(ctx, data.KindFilter.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert kind filter to Terraform value, got error: %s", err))
		return
	}

	workloadNames, err := getStringList(ctx, data.WorkloadNames.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert workload name to Terraform value, got error: %s", err))
		return
	}

	nodeGroupNames, err := getStringList(ctx, data.NodeGroupNames.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert node group name to Terraform value, got error: %s", err))
		return
	}

	clusterIds, err := getStringList(ctx, data.ClusterIds.Elements())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert cluster ID to Terraform value, got error: %s", err))
		return
	}

	namespaceSelector, err := data.NamespaceSelector.toProto(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert namespace selector to Terraform value, got error: %s", err))
		return
	}
	workloadSelector, err := data.WorkloadSelector.toProto(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert workload selector to Terraform value, got error: %s", err))
		return
	}

	updateWorkloadPolicyTargetReq := &apiv1.UpdateWorkloadPolicyTargetRequest{
		TeamId:            r.client.TeamId,
		TargetId:          data.Id.ValueString(),
		PolicyId:          data.PolicyId.ValueStringPointer(),
		Name:              data.Name.ValueString(),
		Description:       data.Description.ValueString(),
		Priority:          data.Priority.ValueInt32(),
		Enabled:           data.Enabled.ValueBool(),
		NamespaceSelector: namespaceSelector,
		WorkloadSelector:  workloadSelector,
		KindFilter:        kindFilters,
		NamePattern:       data.NamePattern.toProto(),
		WorkloadNames:     workloadNames,
		NodeGroupNames:    nodeGroupNames,
		ClusterIds:        clusterIds,
	}

	updateWorkloadPolicyTargetResp, err := r.client.RecommendationClient.UpdateWorkloadPolicyTarget(ctx, connect.NewRequest(updateWorkloadPolicyTargetReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster, got error: %s", err))
		return
	}

	if updateWorkloadPolicyTargetResp.Msg.Target == nil {
		resp.Diagnostics.AddError("Client Error", "Cluster not updated")
		return
	}

	data.fromProto(updateWorkloadPolicyTargetResp.Msg.Target)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadPolicyTargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkloadPolicyTargetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteWorkloadPolicyTargetReq := &apiv1.DeleteWorkloadPolicyTargetRequest{
		TeamId:    r.client.TeamId,
		TargetIds: []string{data.Id.ValueString()},
	}

	_, err := r.client.RecommendationClient.DeleteWorkloadPolicyTarget(ctx, connect.NewRequest(deleteWorkloadPolicyTargetReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster, got error: %s", err))
		return
	}
}

func (r *WorkloadPolicyTargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (l *LabelSelector) toProto(ctx context.Context) (*apiv1.LabelSelector, error) {
	if l == nil {
		return nil, nil
	}
	matchLabels, err := getStringMap(ctx, l.MatchLabels.Elements())
	if err != nil {
		return nil, err
	}

	// Manually extract match expressions from types.Object (can't use getElementList due to nested types.List)
	var matchExpressions []*apiv1.LabelSelectorRequirement
	for _, elem := range l.MatchExpressions.Elements() {
		objVal, ok := elem.(types.Object)
		if !ok {
			continue
		}

		attrs := objVal.Attributes()

		key := ""
		if keyAttr, ok := attrs["key"].(types.String); ok && !keyAttr.IsNull() {
			key = keyAttr.ValueString()
		}

		operatorStr := ""
		if opAttr, ok := attrs["operator"].(types.String); ok && !opAttr.IsNull() {
			operatorStr = opAttr.ValueString()
		}

		var operator apiv1.LabelSelectorOperator
		switch operatorStr {
		case "In":
			operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_IN
		case "NotIn":
			operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_NOT_IN
		case "Exists":
			operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_EXISTS
		case "DoesNotExist":
			operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_DOES_NOT_EXIST
		case "Gt":
			operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_GT
		case "Lt":
			operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_LT
		}

		values := []string{}
		if valuesAttr, ok := attrs["values"].(types.List); ok && !valuesAttr.IsNull() {
			values, err = getStringList(ctx, valuesAttr.Elements())
			if err != nil {
				return nil, err
			}
		}

		matchExpressions = append(matchExpressions, &apiv1.LabelSelectorRequirement{
			Key:      key,
			Operator: operator,
			Values:   values,
		})
	}

	return &apiv1.LabelSelector{
		MatchLabels:      matchLabels,
		MatchExpressions: matchExpressions,
	}, nil
}

func (m *MatchExpression) toProto(ctx context.Context) (*apiv1.LabelSelectorRequirement, error) {
	var operator apiv1.LabelSelectorOperator
	switch m.Operator.ValueString() {
	case "In":
		operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_IN
	case "NotIn":
		operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_NOT_IN
	case "Exists":
		operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_EXISTS
	case "DoesNotExist":
		operator = apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_DOES_NOT_EXIST
	}

	values, err := getStringList(ctx, m.Values.Elements())
	if err != nil {
		return nil, err
	}

	return &apiv1.LabelSelectorRequirement{
		Key:      m.Key.ValueString(),
		Operator: operator,
		Values:   values,
	}, nil
}

func (r *RegexPattern) toProto() *apiv1.RegexPattern {
	if r == nil {
		return nil
	}
	return &apiv1.RegexPattern{
		Pattern: r.Pattern.ValueString(),
		Flags:   r.Flags.ValueString(),
	}
}

func getKindFilters(ctx context.Context, values []attr.Value) ([]apiv1.K8SObjectKind, error) {
	return getElementList(ctx, values, func(ctx context.Context, value string) (apiv1.K8SObjectKind, error) {
		switch value {
		case "Pod":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_POD, nil
		case "Job":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_JOB, nil
		case "Deployment":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_DEPLOYMENT, nil
		case "StatefulSet":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_STATEFUL_SET, nil
		case "DaemonSet":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_DAEMON_SET, nil
		case "ReplicaSet":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_REPLICA_SET, nil
		case "CronJob":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_CRON_JOB, nil
		case "ReplicationController":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_REPLICATION_CONTROLLER, nil
		case "Rollout":
			return apiv1.K8SObjectKind_K8S_OBJECT_KIND_ARGO_ROLLOUT, nil
		}
		return apiv1.K8SObjectKind_K8S_OBJECT_KIND_UNSPECIFIED, fmt.Errorf("invalid kind: %s", value)
	})
}

func fromKindFilter(kinds []apiv1.K8SObjectKind) []attr.Value {
	var kindsList []attr.Value
	for _, kind := range kinds {
		switch kind {
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_POD:
			kindsList = append(kindsList, types.StringValue("Pod"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_JOB:
			kindsList = append(kindsList, types.StringValue("Job"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_DEPLOYMENT:
			kindsList = append(kindsList, types.StringValue("Deployment"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_STATEFUL_SET:
			kindsList = append(kindsList, types.StringValue("StatefulSet"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_DAEMON_SET:
			kindsList = append(kindsList, types.StringValue("DaemonSet"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_REPLICA_SET:
			kindsList = append(kindsList, types.StringValue("ReplicaSet"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_CRON_JOB:
			kindsList = append(kindsList, types.StringValue("CronJob"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_REPLICATION_CONTROLLER:
			kindsList = append(kindsList, types.StringValue("ReplicationController"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_ARGO_ROLLOUT:
			kindsList = append(kindsList, types.StringValue("Rollout"))
		case apiv1.K8SObjectKind_K8S_OBJECT_KIND_UNSPECIFIED:
			kindsList = append(kindsList, types.StringValue("Unspecified"))
		}
	}
	return kindsList
}

func (m *WorkloadPolicyTargetResourceModel) fromProto(target *apiv1.WorkloadPolicyTarget) {
	m.Id = types.StringValue(target.TargetId)
	m.PolicyId = types.StringValue(target.PolicyId)
	m.Name = types.StringValue(target.Name)
	m.Description = types.StringValue(target.Description)
	m.Priority = types.Int32Value(target.Priority)
	m.Enabled = types.BoolValue(target.Enabled)
	m.NamespaceSelector.fromProto(target.NamespaceSelector)
	m.WorkloadSelector.fromProto(target.WorkloadSelector)
	m.KindFilter = types.ListValueMust(types.StringType, fromKindFilter(target.KindFilter))
	m.NamePattern.fromProto(target.NamePattern)
	m.WorkloadNames = types.ListValueMust(types.StringType, fromStringList(target.WorkloadNames))
	m.NodeGroupNames = types.ListValueMust(types.StringType, fromStringList(target.NodeGroupNames))
	m.ClusterIds = types.ListValueMust(types.StringType, fromStringList(target.ClusterIds))
}

func (m *LabelSelector) fromProto(selector *apiv1.LabelSelector) {
	if selector == nil {
		m = nil
		return
	}
	if m == nil {
		m = &LabelSelector{}
	}

	// Handle match_labels: if empty, set to null instead of empty map
	if len(selector.MatchLabels) == 0 {
		m.MatchLabels = types.MapNull(types.StringType)
	} else {
		m.MatchLabels = types.MapValueMust(types.StringType, fromStringMap(selector.MatchLabels))
	}

	// Manually convert match expressions from proto to Terraform types
	var matchExpressions []attr.Value
	for _, expr := range selector.MatchExpressions {
		// Convert operator enum to string
		var operatorStr string
		switch expr.Operator {
		case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_IN:
			operatorStr = "In"
		case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_NOT_IN:
			operatorStr = "NotIn"
		case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_EXISTS:
			operatorStr = "Exists"
		case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_DOES_NOT_EXIST:
			operatorStr = "DoesNotExist"
		case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_GT:
			operatorStr = "Gt"
		case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_LT:
			operatorStr = "Lt"
		}

		// Convert values to Terraform list
		values := types.ListValueMust(types.StringType, fromStringList(expr.Values))

		// Build the match expression object
		matchExpr := types.ObjectValueMust(
			MatchExpression{}.AttrTypes(),
			map[string]attr.Value{
				"key":      types.StringValue(expr.Key),
				"operator": types.StringValue(operatorStr),
				"values":   values,
			},
		)
		matchExpressions = append(matchExpressions, matchExpr)
	}

	// Handle match_expressions: if empty, set to null instead of empty list
	if len(matchExpressions) == 0 {
		m.MatchExpressions = types.ListNull(types.ObjectType{AttrTypes: MatchExpression{}.AttrTypes()})
	} else {
		m.MatchExpressions = types.ListValueMust(types.ObjectType{AttrTypes: MatchExpression{}.AttrTypes()}, matchExpressions)
	}
}

func (m *RegexPattern) fromProto(pattern *apiv1.RegexPattern) {
	if pattern == nil {
		m = nil
		return
	}
	if m == nil {
		m = &RegexPattern{}
	}
	m.Pattern = types.StringValue(pattern.Pattern)
	m.Flags = types.StringValue(pattern.Flags)
}
