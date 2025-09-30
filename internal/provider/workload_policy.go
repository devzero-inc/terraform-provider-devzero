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
	MinChangePercent        types.Float32             `tfsdk:"min_change_percent"`
	MinDataPoints           types.Int32               `tfsdk:"min_data_points"`
	StabilityCvMax          types.Float32             `tfsdk:"stability_cv_max"`
	HysteresisVsTarget      types.Float32             `tfsdk:"hysteresis_vs_target"`
	DriftDeltaPercent       types.Float32             `tfsdk:"drift_delta_percent"`
	MinVpaWindowDataPoints  types.Int32               `tfsdk:"min_vpa_window_data_points"`
	CooldownMinutes         types.Int32               `tfsdk:"cooldown_minutes"`
}

type VerticalScalingOptions struct {
	Enabled                 types.Bool    `tfsdk:"enabled"`
	MinRequest              types.Int64   `tfsdk:"min_request"`
	MaxRequest              types.Int64   `tfsdk:"max_request"`
	OverheadMultiplier      types.Float32 `tfsdk:"overhead_multiplier"`
	LimitsAdjustmentEnabled types.Bool    `tfsdk:"limits_adjustment_enabled"`
	TargetPercentile        types.Float32 `tfsdk:"target_percentile"`
	MaxScaleUpPercent       types.Float32 `tfsdk:"max_scale_up_percent"`
	MaxScaleDownPercent     types.Float32 `tfsdk:"max_scale_down_percent"`
	LimitMultiplier         types.Float32 `tfsdk:"limit_multiplier"`
	MinDataPoints           types.Int32   `tfsdk:"min_data_points"`
}

type HorizontalScalingOptions struct {
	Enabled                 types.Bool    `tfsdk:"enabled"`
	MinReplicas             types.Int32   `tfsdk:"min_replicas"`
	MaxReplicas             types.Int32   `tfsdk:"max_replicas"`
	TargetUtilization       types.Float32 `tfsdk:"target_utilization"`
	PrimaryMetric           types.String  `tfsdk:"primary_metric"`
	MinDataPoints           types.Int32   `tfsdk:"min_data_points"`
	MaxReplicaChangePercent types.Float32 `tfsdk:"max_replica_change_percent"`
}

func (r *WorkloadPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_policy"
}

func (r *WorkloadPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	verticalScalingAttributes := func(defaultEnabled bool) map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description:         "Enable or disable vertical scaling for this resource",
				MarkdownDescription: "Enable or disable vertical scaling for this resource. When disabled, vertical recommendations will not be applied.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(defaultEnabled),
			},
			"min_request": schema.Int64Attribute{
				Description:         "Lower bound for container resource requests",
				MarkdownDescription: "Lower bound for container resource requests (e.g., CPU millicores or memory bytes) considered by the recommender.",
				Optional:            true,
			},
			"max_request": schema.Int64Attribute{
				Description:         "Upper bound for container resource requests",
				MarkdownDescription: "Upper bound for container resource requests (e.g., CPU millicores or memory bytes) considered by the recommender.",
				Optional:            true,
			},
			"overhead_multiplier": schema.Float32Attribute{
				Description:         "Additional headroom added to recommendations",
				MarkdownDescription: "Additional headroom added to recommendations, expressed as a fraction (e.g., 0.05 for 5%).",
				Optional:            true,
				Computed:            true,
				Default:             float32default.StaticFloat32(0.05),
			},
			"limits_adjustment_enabled": schema.BoolAttribute{
				Description:         "Allow recommender to adjust container limits as well as requests",
				MarkdownDescription: "Allow recommender to adjust container limits as well as requests. When disabled, only requests are modified.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"target_percentile": schema.Float32Attribute{
				Description:         "Target percentile for resource sizing (0.0-1.0)",
				MarkdownDescription: "Target percentile for resource sizing (e.g., 0.75 = P75).",
				Optional:            true,
			},
			"max_scale_up_percent": schema.Float32Attribute{
				Description: "Maximum percent to scale up in one step",
				Optional:    true,
			},
			"max_scale_down_percent": schema.Float32Attribute{
				Description: "Maximum percent to scale down in one step",
				Optional:    true,
			},
			"limit_multiplier": schema.Float32Attribute{
				Description:         "How much higher limits should be vs requests",
				MarkdownDescription: "How much higher limits should be vs requests (e.g., 2.0 = 2x the request).",
				Optional:            true,
			},
			"min_data_points": schema.Int32Attribute{
				Description: "Minimum data points required for VPA decisions",
				Optional:    true,
			},
		}
	}

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Configures DevZero workload recommendation policies, including triggers, scaling targets, and scheduler options.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Unique identifier of the workload policy",
				MarkdownDescription: "Unique identifier of the workload policy. Managed by the provider.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "Human-friendly name for the policy",
				MarkdownDescription: "Human-friendly name for the policy. Used for display in the DevZero UI.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Free-form description of the policy",
				MarkdownDescription: "Free-form description of the policy to help others understand its intent and scope.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"action_triggers": schema.ListAttribute{
				Description: "When to apply this policy",
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
				Description:         "Cron expression for scheduled application",
				MarkdownDescription: "Cron expression for scheduled application. Uses standard 5-field cron format in the cluster timezone.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("*/15 * * * *"),
			},
			"detection_triggers": schema.ListAttribute{
				Description: "Events that trigger application of this policy",
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
			"loopback_period_seconds": schema.Int32Attribute{
				Description:         "Window of historical data for recommendations",
				MarkdownDescription: "Loopback period seconds of the workload policy. The loopback period is the period of time to look back for resource usage data.",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(86400),
			},
			"startup_period_seconds": schema.Int32Attribute{
				Description:         "Ignore early-life metrics for this duration",
				MarkdownDescription: "Startup period seconds of the workload policy. The startup period is the period of time to ignore resource usage data after the workload is started.",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(0),
			},
			"cpu_vertical_scaling": schema.SingleNestedAttribute{
				Description: "CPU vertical scaling options",
				Optional:    true,
				Attributes:  verticalScalingAttributes(true),
			},
			"memory_vertical_scaling": schema.SingleNestedAttribute{
				Description: "Memory vertical scaling options",
				Optional:    true,
				Attributes:  verticalScalingAttributes(true),
			},
			"gpu_vertical_scaling": schema.SingleNestedAttribute{
				Description: "GPU vertical scaling options",
				Optional:    true,
				Attributes:  verticalScalingAttributes(false),
			},
			"gpu_vram_vertical_scaling": schema.SingleNestedAttribute{
				Description: "GPU VRAM vertical scaling options",
				Optional:    true,
				Attributes:  verticalScalingAttributes(false),
			},
			"horizontal_scaling": schema.SingleNestedAttribute{
				Description: "Horizontal scaling options",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Enable or disable horizontal scaling",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"min_replicas": schema.Int32Attribute{
						Description: "Lower bound on replicas",
						Optional:    true,
					},
					"max_replicas": schema.Int32Attribute{
						Description: "Upper bound on replicas",
						Optional:    true,
					},
					"target_utilization": schema.Float32Attribute{
						Description: "Target utilization for primary metric (0.0-1.0)",
						Optional:    true,
					},
					"primary_metric": schema.StringAttribute{
						Description: "Primary metric to use for HPA decisions",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("cpu", "memory", "gpu", "network"),
						},
					},
					"min_data_points": schema.Int32Attribute{
						Description: "Minimum data points required for HPA decisions",
						Optional:    true,
					},
					"max_replica_change_percent": schema.Float32Attribute{
						Description: "Maximum percent replica change in one step",
						Optional:    true,
					},
				},
			},
			"live_migration_enabled": schema.BoolAttribute{
				Description: "Allow live migration when applying recommendations",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"scheduler_plugins": schema.ListAttribute{
				Description: "Kubernetes scheduler plugins to activate",
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
				Description:         "Cron expression for background defragmentation",
				MarkdownDescription: "Cron expression for background defragmentation that can move workloads to reduce fragmentation.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("*/15 * * * *"),
			},
			"min_change_percent": schema.Float32Attribute{
				Description: "Global minimum change threshold for applying recommendations",
				Optional:    true,
			},
			"min_data_points": schema.Int32Attribute{
				Description: "Global minimum data points required for recommendations",
				Optional:    true,
			},
			"stability_cv_max": schema.Float32Attribute{
				Description: "Maximum coefficient of variation to consider stable",
				Optional:    true,
			},
			"hysteresis_vs_target": schema.Float32Attribute{
				Description: "Hysteresis threshold vs target for HPA coordination",
				Optional:    true,
			},
			"drift_delta_percent": schema.Float32Attribute{
				Description: "Percentage drift from baseline that triggers VPA refresh",
				Optional:    true,
			},
			"min_vpa_window_data_points": schema.Int32Attribute{
				Description: "Minimum data points in VPA analysis window",
				Optional:    true,
			},
			"cooldown_minutes": schema.Int32Attribute{
				Description: "Minutes to wait between applying recommendations",
				Optional:    true,
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
		TargetPercentile:        o.TargetPercentile.ValueFloat32Pointer(),
		MaxScaleUpPercent:       o.MaxScaleUpPercent.ValueFloat32Pointer(),
		MaxScaleDownPercent:     o.MaxScaleDownPercent.ValueFloat32Pointer(),
		LimitMultiplier:         o.LimitMultiplier.ValueFloat32Pointer(),
		MinDataPoints:           o.MinDataPoints.ValueInt32Pointer(),
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
	if target.TargetPercentile != nil {
		o.TargetPercentile = types.Float32Value(*target.TargetPercentile)
	}
	if target.MaxScaleUpPercent != nil {
		o.MaxScaleUpPercent = types.Float32Value(*target.MaxScaleUpPercent)
	}
	if target.MaxScaleDownPercent != nil {
		o.MaxScaleDownPercent = types.Float32Value(*target.MaxScaleDownPercent)
	}
	if target.LimitMultiplier != nil {
		o.LimitMultiplier = types.Float32Value(*target.LimitMultiplier)
	}
	if target.MinDataPoints != nil {
		o.MinDataPoints = types.Int32Value(*target.MinDataPoints)
	}
}

func (o *HorizontalScalingOptions) toProto() *apiv1.HorizontalScalingOptimizationTarget {
	if o == nil {
		return nil
	}
	return &apiv1.HorizontalScalingOptimizationTarget{
		Enabled:                 o.Enabled.ValueBool(),
		MinReplicas:             o.MinReplicas.ValueInt32Pointer(),
		MaxReplicas:             o.MaxReplicas.ValueInt32Pointer(),
		TargetUtilization:       o.TargetUtilization.ValueFloat32Pointer(),
		PrimaryMetric:           o.toHPAMetric(),
		MinDataPoints:           o.MinDataPoints.ValueInt32Pointer(),
		MaxReplicaChangePercent: o.MaxReplicaChangePercent.ValueFloat32Pointer(),
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
	if target.TargetUtilization != nil {
		o.TargetUtilization = types.Float32Value(*target.TargetUtilization)
	}
	o.PrimaryMetric = types.StringValue(o.fromHPAMetric(target.GetPrimaryMetric()))
	if target.MinDataPoints != nil {
		o.MinDataPoints = types.Int32Value(*target.MinDataPoints)
	}
	if target.MaxReplicaChangePercent != nil {
		o.MaxReplicaChangePercent = types.Float32Value(*target.MaxReplicaChangePercent)
	}
}

func (o *HorizontalScalingOptions) toHPAMetric() *apiv1.HPAMetricType {
	var metric apiv1.HPAMetricType
	switch o.PrimaryMetric.ValueString() {
	case "cpu":
		metric = apiv1.HPAMetricType_HPA_METRIC_TYPE_CPU
	case "memory":
		metric = apiv1.HPAMetricType_HPA_METRIC_TYPE_MEMORY
	case "gpu":
		metric = apiv1.HPAMetricType_HPA_METRIC_TYPE_GPU
	case "network":
		metric = apiv1.HPAMetricType_HPA_METRIC_TYPE_NETWORK
	default:
		return nil
	}
	return &metric
}

func (o *HorizontalScalingOptions) fromHPAMetric(metric apiv1.HPAMetricType) string {
	switch metric {
	case apiv1.HPAMetricType_HPA_METRIC_TYPE_CPU:
		return "cpu"
	case apiv1.HPAMetricType_HPA_METRIC_TYPE_MEMORY:
		return "memory"
	case apiv1.HPAMetricType_HPA_METRIC_TYPE_GPU:
		return "gpu"
	case apiv1.HPAMetricType_HPA_METRIC_TYPE_NETWORK:
		return "network"
	default:
		return ""
	}
}
