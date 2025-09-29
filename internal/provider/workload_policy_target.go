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
	Id                 types.String   `tfsdk:"id"`
	PolicyId           types.String   `tfsdk:"policy_id"`
	Name               types.String   `tfsdk:"name"`
	Description        types.String   `tfsdk:"description"`
	Priority           types.Int32    `tfsdk:"priority"`
	Enabled            types.Bool     `tfsdk:"enabled"`
	NamespaceSelector  *LabelSelector `tfsdk:"namespace_selector"`
	WorkloadSelector   *LabelSelector `tfsdk:"workload_selector"`
	KindFilter         types.List     `tfsdk:"kind_filter"`
	NamePattern        *RegexPattern  `tfsdk:"name_pattern"`
	AnnotationSelector *LabelSelector `tfsdk:"annotation_selector"`
	WorkloadNames      types.List     `tfsdk:"workload_names"`
	NodeGroupNames     types.List     `tfsdk:"node_group_names"`
	ClusterIds         types.List     `tfsdk:"cluster_ids"`
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
			Description: "Match labels of the label selector",
			Optional:    true,
			ElementType: types.StringType,
		},
		"match_expressions": schema.ListNestedAttribute{
			Description: "Match expressions of the label selector",
			Optional:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"key": schema.StringAttribute{
						Description: "Key of the match expression",
						Optional:    true,
					},
					"operator": schema.StringAttribute{
						Description: "Operator of the match expression",
						Optional:    true,
					},
					"values": schema.ListAttribute{
						Description: "Values of the match expression",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}

	regexPatternAttributes := map[string]schema.Attribute{
		"pattern": schema.StringAttribute{
			Description: "Pattern of the regex pattern",
			Optional:    true,
		},
		"flags": schema.StringAttribute{
			Description: "Flags of the regex pattern",
			Optional:    true,
		},
	}

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Workload policy target resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the workload policy target",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_id": schema.StringAttribute{
				Description: "ID of the workload policy",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the workload policy target",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the workload policy target",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"priority": schema.Int32Attribute{
				Description: "Priority of the workload policy target",
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(0),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the workload policy target is enabled",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"namespace_selector": schema.SingleNestedAttribute{
				Description: "Namespace selector of the workload policy target",
				Optional:    true,
				Attributes:  labelSelectorAttributes,
			},
			"workload_selector": schema.SingleNestedAttribute{
				Description: "Workload selector of the workload policy target",
				Optional:    true,
				Attributes:  labelSelectorAttributes,
			},
			"kind_filter": schema.ListAttribute{
				Description: "Kind filter of the workload policy target",
				Optional:    true,
				ElementType: types.StringType,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"name_pattern": schema.SingleNestedAttribute{
				Description: "Name pattern of the workload policy target",
				Optional:    true,
				Attributes:  regexPatternAttributes,
			},
			"annotation_selector": schema.SingleNestedAttribute{
				Description: "Annotation selector of the workload policy target",
				Optional:    true,
				Attributes:  labelSelectorAttributes,
			},
			"workload_names": schema.ListAttribute{
				Description: "Workload names of the workload policy target",
				Optional:    true,
				ElementType: types.StringType,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"node_group_names": schema.ListAttribute{
				Description: "Node group names of the workload policy target",
				Optional:    true,
				ElementType: types.StringType,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"cluster_ids": schema.ListAttribute{
				Description: "Cluster IDs of the workload policy target",
				Required:    true,
				ElementType: types.StringType,
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
	annotationSelector, err := data.AnnotationSelector.toProto(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert annotation selector to Terraform value, got error: %s", err))
		return
	}

	createWorkloadPolicyTargetReq := &apiv1.CreateWorkloadPolicyTargetRequest{
		TeamId:             r.client.TeamId,
		PolicyId:           data.PolicyId.ValueString(),
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		Priority:           data.Priority.ValueInt32(),
		Enabled:            data.Enabled.ValueBool(),
		NamespaceSelector:  namespaceSelector,
		WorkloadSelector:   workloadSelector,
		KindFilter:         kindFilters,
		NamePattern:        data.NamePattern.toProto(),
		AnnotationSelector: annotationSelector,
		WorkloadNames:      workloadNames,
		NodeGroupNames:     nodeGroupNames,
		ClusterIds:         clusterIds,
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

	// nodeGroupNames, err := getStringList(ctx, data.NodeGroupNames.Elements())
	// if err != nil {
	// 	resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert node group name to Terraform value, got error: %s", err))
	// 	return
	// }

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
	annotationSelector, err := data.AnnotationSelector.toProto(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert annotation selector to Terraform value, got error: %s", err))
		return
	}

	updateWorkloadPolicyTargetReq := &apiv1.UpdateWorkloadPolicyTargetRequest{
		TeamId:             r.client.TeamId,
		TargetId:           data.Id.ValueString(),
		PolicyId:           data.PolicyId.ValueStringPointer(),
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		Priority:           data.Priority.ValueInt32(),
		Enabled:            data.Enabled.ValueBool(),
		NamespaceSelector:  namespaceSelector,
		WorkloadSelector:   workloadSelector,
		KindFilter:         kindFilters,
		NamePattern:        data.NamePattern.toProto(),
		AnnotationSelector: annotationSelector,
		WorkloadNames:      workloadNames,
		//NodeGroupNames:     nodeGroupNames,
		ClusterIds: clusterIds,
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

	matchExpressions, err := getElementList(ctx, l.MatchExpressions.Elements(), func(ctx context.Context, value *MatchExpression) (*apiv1.LabelSelectorRequirement, error) {
		return value.toProto(ctx)
	})
	if err != nil {
		return nil, err
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
	m.AnnotationSelector.fromProto(target.AnnotationSelector)
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
	m.MatchLabels = types.MapValueMust(types.StringType, fromStringMap(selector.MatchLabels))
	m.MatchExpressions = types.ListValueMust(types.StringType, fromElementList(selector.MatchExpressions, MatchExpression{}.AttrTypes()))
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
