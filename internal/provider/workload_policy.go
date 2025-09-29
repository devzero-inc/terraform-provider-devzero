package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkloadPolicyResource{}
var _ resource.ResourceWithConfigure = &WorkloadPolicyResource{}
var _ resource.ResourceWithImportState = &WorkloadPolicyResource{}

func NewWorkloadPolicyResource() resource.Resource {
	return &WorkloadPolicyResource{}
}

// ExampleResource defines the resource implementation.
type WorkloadPolicyResource struct {
	client *ClientSet
}

// ExampleResourceModel describes the resource data model.
type WorkloadPolicyResourceModel struct {
	Id                      types.String              `tfsdk:"id"`
	Name                    types.String              `tfsdk:"name"`
	Description             types.String              `tfsdk:"description"`
	ActionTriggers          types.List                `tfsdk:"action_triggers"`
	CronSchedule            types.String              `tfsdk:"cron_schedule"`
	DetectionTriggers       types.List                `tfsdk:"detection_triggers"`
	RecommendationMode      types.String              `tfsdk:"recommendation_mode"`
	LoopbackPeriodSeconds   types.Int32               `tfsdk:"loopback_period_seconds"`
	StartupPeriodSeconds    types.Int32               `tfsdk:"startup_period_seconds"`
	CPUVerticalScaling      *VerticalScalingOptions   `tfsdk:"cpu_vertical_scaling"`
	MemoryVerticalScaling   *VerticalScalingOptions   `tfsdk:"memory_vertical_scaling"`
	GPUVerticalScaling      *VerticalScalingOptions   `tfsdk:"gpu_vertical_scaling"`
	GPUVRAMVerticalScaling  *VerticalScalingOptions   `tfsdk:"gpu_vram_vertical_scaling"`
	HorizontalScaling       *HorizontalScalingOptions `tfsdk:"horizontal_scaling"`
	LiveMigrationEnabled    types.Bool                `tfsdk:"live_migration_enabled"`
	SchedulerPlugins        types.List                `tfsdk:"scheduler_plugins"`
	DefragmentationSchedule types.String              `tfsdk:"defragmentation_schedule"`
}

type VerticalScalingOptions struct {
	Enabled                 types.Bool    `tfsdk:"enabled"`
	MinRequest              types.Int64   `tfsdk:"min_request"`
	MaxRequest              types.Int64   `tfsdk:"max_request"`
	OverheadMultiplier      types.Float32 `tfsdk:"overhead_multiplier"`
	LimitsAdjustmentEnabled types.Bool    `tfsdk:"limits_adjustment_enabled"`
}

type HorizontalScalingOptions struct {
	Enabled     types.Bool  `tfsdk:"enabled"`
	MinReplicas types.Int32 `tfsdk:"min_replicas"`
	MaxReplicas types.Int32 `tfsdk:"max_replicas"`
}

func (r *WorkloadPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_policy"
}

func (r *WorkloadPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	verticalScalingAttributes := func(defaultEnabled bool) map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether vertical scaling is enabled for the workload policy",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultEnabled),
			},
			"min_request": schema.Int64Attribute{
				Description: "Minimum request for vertical scaling",
				Optional:    true,
			},
			"max_request": schema.Int64Attribute{
				Description: "Maximum request for vertical scaling",
				Optional:    true,
			},
			"overhead_multiplier": schema.Float32Attribute{
				Description: "Overhead multiplier for vertical scaling",
				Optional:    true,
				Computed:    true,
				Default:     float32default.StaticFloat32(0.05),
			},
			"limits_adjustment_enabled": schema.BoolAttribute{
				Description: "Whether limits adjustment is enabled for vertical scaling",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		}
	}

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Workload policy resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the workload policy",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the workload policy",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the workload policy",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"action_triggers": schema.ListAttribute{
				Description: "Action triggers for when to apply the workload policy",
				MarkdownDescription: "Action triggers for when to apply the workload policy. Only one of `on_schedule` or `on_detection` is allowed." +
					"The `on_schedule` trigger is used to apply the workload policy on a schedule configured with the `cron_schedule` attribute." +
					"The `on_detection` trigger is used to apply the workload policy when a detection trigger event occurs, configured with the `detection_triggers` attribute.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.NoNullValues(),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("on_schedule", "on_detection")),
				},
			},
			"cron_schedule": schema.StringAttribute{
				Description: "Cron schedule to trigger the workload policy.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*/15 * * * *"),
			},
			"detection_triggers": schema.ListAttribute{
				Description: "Detection triggers for when to apply the workload policy",
				MarkdownDescription: "Detection triggers for when to apply the workload policy. Only one of `pod_creation` or `pod_update` is allowed." +
					"The `pod_creation` trigger is used to apply the workload policy when a pod is created." +
					"The `pod_update` trigger is used to apply the workload policy when a pod is updated.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(
					types.ListValueMust(
						types.StringType,
						[]attr.Value{types.StringValue("pod_creation"), types.StringValue("pod_update")},
					),
				),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.NoNullValues(),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("pod_creation", "pod_update")),
				},
			},
			"recommendation_mode": schema.StringAttribute{
				Description: "Recommendation mode of the workload policy",
				MarkdownDescription: "Recommendation mode of the workload policy. The `balanced` mode is the default mode and is used to balance the recommendation between the other modes." +
					"The `aggressive` mode is used to recommend the most aggressive optimization targets." +
					"The `conservative` mode is used to recommend the most conservative optimization targets.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("balanced"),
				Validators: []validator.String{
					stringvalidator.OneOf("balanced", "aggressive", "conservative"),
				},
			},
			"loopback_period_seconds": schema.Int32Attribute{
				Description:         "Loopback period seconds of the workload policy",
				MarkdownDescription: "Loopback period seconds of the workload policy. The loopback period is the period of time to look back for resource usage data.",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(86400),
			},
			"startup_period_seconds": schema.Int32Attribute{
				Description:         "Startup period seconds of the workload policy",
				MarkdownDescription: "Startup period seconds of the workload policy. The startup period is the period of time to ignore resource usage data after the workload is started.",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(0),
			},
			"cpu_vertical_scaling": schema.SingleNestedAttribute{
				Description: "CPU vertical scaling options for the workload policy",
				Optional:    true,
				Attributes:  verticalScalingAttributes(true),
			},
			"memory_vertical_scaling": schema.SingleNestedAttribute{
				Description: "Memory vertical scaling options for the workload policy",
				Optional:    true,
				Attributes:  verticalScalingAttributes(true),
			},
			"gpu_vertical_scaling": schema.SingleNestedAttribute{
				Description: "GPU vertical scaling options for the workload policy",
				Optional:    true,
				Attributes:  verticalScalingAttributes(false),
			},
			"gpu_vram_vertical_scaling": schema.SingleNestedAttribute{
				Description: "GPU VRAM vertical scaling options for the workload policy",
				Optional:    true,
				Attributes:  verticalScalingAttributes(false),
			},
			"horizontal_scaling": schema.SingleNestedAttribute{
				Description: "Horizontal scaling options for the workload policy",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Whether horizontal scaling is enabled for the workload policy",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"min_replicas": schema.Int32Attribute{
						Description: "Minimum replicas for horizontal scaling",
						Optional:    true,
					},
					"max_replicas": schema.Int32Attribute{
						Description: "Maximum replicas for horizontal scaling",
						Optional:    true,
					},
				},
			},
			"live_migration_enabled": schema.BoolAttribute{
				Description: "Whether live migration is enabled for the workload policy",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"scheduler_plugins": schema.ListAttribute{
				Description: "Scheduler plugins for the workload policy",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(
					types.ListValueMust(
						types.StringType,
						[]attr.Value{types.StringValue("dz-scheduler")},
					),
				),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.NoNullValues(),
					listvalidator.UniqueValues(),
				},
			},
			"defragmentation_schedule": schema.StringAttribute{
				Description: "Defragmentation schedule for the workload policy",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*/15 * * * *"),
			},
		},
	}
}

func (r *WorkloadPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkloadPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkloadPolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := data.toProto(ctx, &resp.Diagnostics, r.client.TeamId)

	createWorkloadPolicyReq := &apiv1.CreateWorkloadRecommendationPolicyRequest{
		TeamId: r.client.TeamId,
		Policy: policy,
	}

	createWorkloadPolicyResp, err := r.client.RecommendationClient.CreateWorkloadRecommendationPolicy(ctx, connect.NewRequest(createWorkloadPolicyReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workload policy, got error: %s", err))
		return
	}
	if createWorkloadPolicyResp.Msg.Policy == nil {
		resp.Diagnostics.AddError("Client Error", "Workload policy not created")
		return
	}

	data.fromProto(createWorkloadPolicyResp.Msg.Policy)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkloadPolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getWorkloadPolicyReq := &apiv1.GetWorkloadRecommendationPolicyRequest{
		TeamId:   r.client.TeamId,
		PolicyId: data.Id.ValueString(),
	}

	getWorkloadPolicyResp, err := r.client.RecommendationClient.GetWorkloadRecommendationPolicy(ctx, connect.NewRequest(getWorkloadPolicyReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workload policy, got error: %s", err))
		return
	}

	if getWorkloadPolicyResp.Msg.Policy == nil {
		resp.Diagnostics.AddError("Client Error", "Workload policy not found")
		return
	}

	data.fromProto(getWorkloadPolicyResp.Msg.Policy)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkloadPolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateWorkloadPolicyReq := &apiv1.UpdateWorkloadRecommendationPolicyRequest{
		TeamId: r.client.TeamId,
		Policy: data.toProto(ctx, &resp.Diagnostics, r.client.TeamId),
	}

	updateWorkloadPolicyResp, err := r.client.RecommendationClient.UpdateWorkloadRecommendationPolicy(ctx, connect.NewRequest(updateWorkloadPolicyReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update workload policy, got error: %s", err))
		return
	}

	if updateWorkloadPolicyResp.Msg.Policy == nil {
		resp.Diagnostics.AddError("Client Error", "Workload policy not updated")
		return
	}

	data.fromProto(updateWorkloadPolicyResp.Msg.Policy)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkloadPolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteWorkloadPolicyReq := &apiv1.DeleteWorkloadRecommendationPolicyRequest{
		TeamId:   r.client.TeamId,
		PolicyId: data.Id.ValueString(),
	}

	_, err := r.client.RecommendationClient.DeleteWorkloadRecommendationPolicy(ctx, connect.NewRequest(deleteWorkloadPolicyReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workload policy, got error: %s", err))
		return
	}
}

func (r *WorkloadPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *WorkloadPolicyResourceModel) toProto(ctx context.Context, diags *diag.Diagnostics, teamId string) *apiv1.WorkloadRecommendationPolicy {
	actionTriggers, err := getElementList(ctx, m.ActionTriggers.Elements(), func(ctx context.Context, value string) (apiv1.ActionTrigger, error) {
		switch value {
		case "on_schedule":
			return apiv1.ActionTrigger_ACTION_TRIGGER_ON_SCHEDULE, nil
		case "on_detection":
			return apiv1.ActionTrigger_ACTION_TRIGGER_ON_DETECTION, nil
		default:
			return apiv1.ActionTrigger_ACTION_TRIGGER_UNSPECIFIED, fmt.Errorf("invalid action trigger: %s", value)
		}
	})
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to convert action triggers to Terraform value, got error: %s", err))
		return nil
	}

	detectionTriggers, err := getElementList(ctx, m.DetectionTriggers.Elements(), func(ctx context.Context, value string) (apiv1.WorkloadDetectionTrigger, error) {
		switch value {
		case "pod_creation":
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION, nil
		case "pod_update":
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_UPDATE, nil
		default:
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_UNSPECIFIED, fmt.Errorf("invalid detection trigger: %s", value)
		}
	})
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to convert detection triggers to Terraform value, got error: %s", err))
		return nil
	}

	var recommendationMode apiv1.RecommendationMode
	switch m.RecommendationMode.ValueString() {
	case "balanced":
		recommendationMode = apiv1.RecommendationMode_RECOMMENDATION_MODE_BALANCED
	case "aggressive":
		recommendationMode = apiv1.RecommendationMode_RECOMMENDATION_MODE_AGGRESSIVE
	case "conservative":
		recommendationMode = apiv1.RecommendationMode_RECOMMENDATION_MODE_CONSERVATIVE
	}

	schedulerPlugins, err := getStringList(ctx, m.SchedulerPlugins.Elements())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to convert scheduler plugins to Terraform value, got error: %s", err))
		return nil
	}

	return &apiv1.WorkloadRecommendationPolicy{
		PolicyId:                m.Id.ValueString(),
		TeamId:                  teamId,
		Name:                    m.Name.ValueString(),
		Description:             m.Description.ValueString(),
		ActionTriggers:          actionTriggers,
		CronSchedule:            m.CronSchedule.ValueStringPointer(),
		DetectionTriggers:       detectionTriggers,
		RecommendationMode:      recommendationMode,
		CpuVerticalScaling:      m.CPUVerticalScaling.toProto(),
		MemoryVerticalScaling:   m.MemoryVerticalScaling.toProto(),
		GpuVerticalScaling:      m.GPUVerticalScaling.toProto(),
		GpuVramVerticalScaling:  m.GPUVRAMVerticalScaling.toProto(),
		HorizontalScaling:       m.HorizontalScaling.toProto(),
		LiveMigrationEnabled:    m.LiveMigrationEnabled.ValueBool(),
		SchedulerPlugins:        schedulerPlugins,
		DefragmentationSchedule: m.DefragmentationSchedule.ValueStringPointer(),
	}
}

func (m *WorkloadPolicyResourceModel) fromProto(policy *apiv1.WorkloadRecommendationPolicy) {
	m.Id = types.StringValue(policy.PolicyId)
	m.Name = types.StringValue(policy.Name)
	m.Description = types.StringValue(policy.Description)

	actionTriggers := make([]attr.Value, 0)
	for _, actionTrigger := range policy.ActionTriggers {
		var trigger types.String
		switch actionTrigger {
		case apiv1.ActionTrigger_ACTION_TRIGGER_ON_SCHEDULE:
			trigger = types.StringValue("on_schedule")
		case apiv1.ActionTrigger_ACTION_TRIGGER_ON_DETECTION:
			trigger = types.StringValue("on_detection")
		}
		actionTriggers = append(actionTriggers, trigger)
	}
	m.ActionTriggers = types.ListValueMust(types.StringType, actionTriggers)

	m.CronSchedule = types.StringValue(*policy.CronSchedule)

	detectionTriggers := make([]attr.Value, 0)
	for _, detectionTrigger := range policy.DetectionTriggers {
		var trigger types.String
		switch detectionTrigger {
		case apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION:
			trigger = types.StringValue("pod_creation")
		case apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_UPDATE:
			trigger = types.StringValue("pod_update")
		}
		detectionTriggers = append(detectionTriggers, trigger)
	}
	m.DetectionTriggers = types.ListValueMust(types.StringType, detectionTriggers)

	var recommendationMode types.String
	switch policy.RecommendationMode {
	case apiv1.RecommendationMode_RECOMMENDATION_MODE_BALANCED:
		recommendationMode = types.StringValue("balanced")
	case apiv1.RecommendationMode_RECOMMENDATION_MODE_AGGRESSIVE:
		recommendationMode = types.StringValue("aggressive")
	case apiv1.RecommendationMode_RECOMMENDATION_MODE_CONSERVATIVE:
		recommendationMode = types.StringValue("conservative")
	}
	m.RecommendationMode = recommendationMode

	m.CPUVerticalScaling.fromProto(policy.CpuVerticalScaling)
	m.MemoryVerticalScaling.fromProto(policy.MemoryVerticalScaling)
	m.GPUVerticalScaling.fromProto(policy.GpuVerticalScaling)
	m.GPUVRAMVerticalScaling.fromProto(policy.GpuVramVerticalScaling)
	m.HorizontalScaling.fromProto(policy.HorizontalScaling)
	m.LiveMigrationEnabled = types.BoolValue(policy.LiveMigrationEnabled)

	var schedulerPlugins []attr.Value
	for _, schedulerPlugin := range policy.SchedulerPlugins {
		schedulerPlugins = append(schedulerPlugins, types.StringValue(schedulerPlugin))
	}
	m.SchedulerPlugins = types.ListValueMust(types.StringType, schedulerPlugins)

	m.DefragmentationSchedule = types.StringValue(*policy.DefragmentationSchedule)

}

func (o *VerticalScalingOptions) toProto() *apiv1.VerticalScalingOptimizationTarget {
	if o == nil {
		return nil
	}
	return &apiv1.VerticalScalingOptimizationTarget{
		Enabled:                 o.Enabled.ValueBool(),
		MinRequest:              o.MinRequest.ValueInt64Pointer(),
		MaxRequest:              o.MaxRequest.ValueInt64Pointer(),
		OverheadMultiplier:      o.OverheadMultiplier.ValueFloat32Pointer(),
		LimitsAdjustmentEnabled: o.LimitsAdjustmentEnabled.ValueBoolPointer(),
	}
}

func (o *VerticalScalingOptions) fromProto(target *apiv1.VerticalScalingOptimizationTarget) {
	if target == nil {
		o = nil
		return
	}
	if o == nil {
		o = &VerticalScalingOptions{}
	}
	o.Enabled = types.BoolValue(target.Enabled)

	if target.MinRequest != nil {
		o.MinRequest = types.Int64Value(*target.MinRequest)
	}
	if target.MaxRequest != nil {
		o.MaxRequest = types.Int64Value(*target.MaxRequest)
	}
	if target.OverheadMultiplier != nil {
		o.OverheadMultiplier = types.Float32Value(*target.OverheadMultiplier)
	}
	if target.LimitsAdjustmentEnabled != nil {
		o.LimitsAdjustmentEnabled = types.BoolValue(*target.LimitsAdjustmentEnabled)
	}
}

func (o *HorizontalScalingOptions) toProto() *apiv1.HorizontalScalingOptimizationTarget {
	if o == nil {
		return nil
	}
	return &apiv1.HorizontalScalingOptimizationTarget{
		Enabled:     o.Enabled.ValueBool(),
		MinReplicas: o.MinReplicas.ValueInt32Pointer(),
		MaxReplicas: o.MaxReplicas.ValueInt32Pointer(),
	}
}

func (o *HorizontalScalingOptions) fromProto(target *apiv1.HorizontalScalingOptimizationTarget) {
	if target == nil {
		o = nil
		return
	}
	if o == nil {
		o = &HorizontalScalingOptions{}
	}
	o.Enabled = types.BoolValue(target.Enabled)
	if target.MinReplicas != nil {
		o.MinReplicas = types.Int32Value(*target.MinReplicas)
	}
	if target.MaxReplicas != nil {
		o.MaxReplicas = types.Int32Value(*target.MaxReplicas)
	}
}
