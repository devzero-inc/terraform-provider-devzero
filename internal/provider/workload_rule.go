package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

var _ resource.Resource = &WorkloadRuleResource{}
var _ resource.ResourceWithConfigure = &WorkloadRuleResource{}
var _ resource.ResourceWithImportState = &WorkloadRuleResource{}

func NewWorkloadRuleResource() resource.Resource {
	return &WorkloadRuleResource{}
}

type WorkloadRuleResource struct {
	client *ClientSet
}

type WorkloadRuleResourceModel struct {
	Id                        types.String             `tfsdk:"id"`
	ClusterId                 types.String             `tfsdk:"cluster_id"`
	Namespace                 types.String             `tfsdk:"namespace"`
	Kind                      types.String             `tfsdk:"kind"`
	Name                      types.String             `tfsdk:"name"`
	AutoGenerate              types.Bool               `tfsdk:"auto_generate"`
	CpuRule                   *ResourceRuleConfigModel `tfsdk:"cpu_rule"`
	MemoryRule                *ResourceRuleConfigModel `tfsdk:"memory_rule"`
	GpuRule                   *ResourceRuleConfigModel `tfsdk:"gpu_rule"`
	HpaRule                   *HPARuleConfigModel      `tfsdk:"hpa_rule"`
	EmergencyResponse         *EmergencyResponseModel  `tfsdk:"emergency_response"`
	ActionTriggers            types.List               `tfsdk:"action_triggers"`
	StartupPeriodSeconds      types.Int64              `tfsdk:"startup_period_seconds"`
	CronSchedule              types.String             `tfsdk:"cron_schedule"`
	CooldownMinutes           types.Int32              `tfsdk:"cooldown_minutes"`
	DetectionTriggers         types.List               `tfsdk:"detection_triggers"`
	SchedulerPlugins          types.List               `tfsdk:"scheduler_plugins"`
	DefragmentationSchedule   types.String             `tfsdk:"defragmentation_schedule"`
	LiveMigrationEnabled      types.Bool               `tfsdk:"live_migration_enabled"`
	UseInPlaceVerticalScaling types.Bool               `tfsdk:"use_in_place_vertical_scaling"`
	Containers                []ContainerRuleModel     `tfsdk:"containers"`
	Disabled                  types.Bool               `tfsdk:"disabled"`
	LookbackPeriodSeconds     types.Int32              `tfsdk:"lookback_period_seconds"`

	AllowInPlaceMemoryLimitDecrease types.Bool             `tfsdk:"allow_in_place_memory_limit_decrease"`
	JvmHeapRule                     *JVMHeapRuleModel      `tfsdk:"jvm_heap_rule"`
	JvmCpuStartupFloorMillicores    types.Int64            `tfsdk:"jvm_cpu_startup_floor_millicores"`
	KedaScaledObject                *KEDAScaledObjectModel `tfsdk:"keda_scaled_object"`
}

type ResourceRuleConfigModel struct {
	Enabled                 types.Bool    `tfsdk:"enabled"`
	MinRequest              types.Int64   `tfsdk:"min_request"`
	MaxRequest              types.Int64   `tfsdk:"max_request"`
	LimitMultiplier         types.Float32 `tfsdk:"limit_multiplier"`
	LimitsAdjustmentEnabled types.Bool    `tfsdk:"limits_adjustment_enabled"`
	TargetPercentile        types.Float32 `tfsdk:"target_percentile"`
	MaxScaleUpPercent       types.Float32 `tfsdk:"max_scale_up_percent"`
	MaxScaleDownPercent     types.Float32 `tfsdk:"max_scale_down_percent"`
	LimitsRemovalEnabled    types.Bool    `tfsdk:"limits_removal_enabled"`
	InitialRequest          types.Int64   `tfsdk:"initial_request"`
	FloorPercent            types.Int64   `tfsdk:"floor_percent"`
	CeilingPercent          types.Int64   `tfsdk:"ceiling_percent"`
	InitialLimit            types.Int64   `tfsdk:"initial_limit"`
	LimitFloorPercent       types.Int64   `tfsdk:"limit_floor_percent"`
	LimitCeilingPercent     types.Int64   `tfsdk:"limit_ceiling_percent"`
	RequestUseRss           types.Bool    `tfsdk:"request_use_rss"`
	LimitUseRss             types.Bool    `tfsdk:"limit_use_rss"`
}

type HPARuleConfigModel struct {
	Enabled                  types.Bool              `tfsdk:"enabled"`
	MinReplicas              types.Int32             `tfsdk:"min_replicas"`
	MaxReplicas              types.Int32             `tfsdk:"max_replicas"`
	MaxReplicaChangePercent  types.Float32           `tfsdk:"max_replica_change_percent"`
	ScaleDownCooldownSeconds types.Int32             `tfsdk:"scale_down_cooldown_seconds"`
	Metrics                  []HPAMetricTriggerModel `tfsdk:"metrics"`
	CompositeFormula         types.String            `tfsdk:"composite_formula"`
	Behavior                 *HPABehaviorModel       `tfsdk:"behavior"`
	Fallback                 *HPAFallbackModel       `tfsdk:"fallback"`
}

type HPAMetricTriggerModel struct {
	Type              types.String `tfsdk:"type"`
	TargetUtilization types.String `tfsdk:"target_utilization"`
	TargetValue       types.String `tfsdk:"target_value"`
	Weight            types.String `tfsdk:"weight"`
	Metadata          types.Map    `tfsdk:"metadata"`
	ServerAddress     types.String `tfsdk:"server_address"`
	Query             types.String `tfsdk:"query"`
	ConnectorId       types.String `tfsdk:"connector_id"`
}

type HPAFallbackModel struct {
	Replicas         types.Int32  `tfsdk:"replicas"`
	Behavior         types.String `tfsdk:"behavior"`
	FailureThreshold types.Int32  `tfsdk:"failure_threshold"`
}

type HPAScalingPolicyModel struct {
	Type          types.String `tfsdk:"type"`
	Value         types.Int32  `tfsdk:"value"`
	PeriodSeconds types.Int32  `tfsdk:"period_seconds"`
}

type HPAScalingRulesModel struct {
	StabilizationWindowSeconds types.Int32             `tfsdk:"stabilization_window_seconds"`
	SelectPolicy               types.String            `tfsdk:"select_policy"`
	Policies                   []HPAScalingPolicyModel `tfsdk:"policies"`
}

type HPABehaviorModel struct {
	ScaleUp   *HPAScalingRulesModel `tfsdk:"scale_up"`
	ScaleDown *HPAScalingRulesModel `tfsdk:"scale_down"`
}

type EmergencyResponseModel struct {
	OomEnabled              types.Bool    `tfsdk:"oom_enabled"`
	OomMemoryMultiplier     types.Float32 `tfsdk:"oom_memory_multiplier"`
	OomMaxReactions         types.Int32   `tfsdk:"oom_max_reactions"`
	OomCooldownSeconds      types.Int32   `tfsdk:"oom_cooldown_seconds"`
	CpuThrottlingEnabled    types.Bool    `tfsdk:"cpu_throttling_enabled"`
	CpuThrottlingThreshold  types.Float32 `tfsdk:"cpu_throttling_threshold"`
	CpuThrottlingMultiplier types.Float32 `tfsdk:"cpu_throttling_multiplier"`
}

type JVMHeapRuleModel struct {
	Enabled                types.Bool    `tfsdk:"enabled"`
	TargetPercentile       types.Float32 `tfsdk:"target_percentile"`
	HeadroomMultiplier     types.Float32 `tfsdk:"headroom_multiplier"`
	NonHeapOverheadPercent types.Float32 `tfsdk:"non_heap_overhead_percent"`
	NonHeapOverheadBytes   types.Int64   `tfsdk:"non_heap_overhead_bytes"`
	MinHeapBytes           types.Int64   `tfsdk:"min_heap_bytes"`
	MaxHeapBytes           types.Int64   `tfsdk:"max_heap_bytes"`
	PreferContainerSupport types.Bool    `tfsdk:"prefer_container_support"`
}

type KEDAScaledObjectModel struct {
	Triggers              []KEDATriggerModel `tfsdk:"triggers"`
	MinReplicaCount       types.Int32        `tfsdk:"min_replica_count"`
	MaxReplicaCount       types.Int32        `tfsdk:"max_replica_count"`
	IdleReplicaCount      types.Int32        `tfsdk:"idle_replica_count"`
	PollingInterval       types.Int32        `tfsdk:"polling_interval"`
	CooldownPeriod        types.Int32        `tfsdk:"cooldown_period"`
	InitialCooldownPeriod types.Int32        `tfsdk:"initial_cooldown_period"`
	Fallback              *KEDAFallbackModel `tfsdk:"fallback"`
	Advanced              *KEDAAdvancedModel `tfsdk:"advanced"`
}

type KEDATriggerModel struct {
	Type              types.String                `tfsdk:"type"`
	Name              types.String                `tfsdk:"name"`
	Metadata          types.Map                   `tfsdk:"metadata"`
	MetricType        types.String                `tfsdk:"metric_type"`
	AuthenticationRef *KEDAAuthenticationRefModel `tfsdk:"authentication_ref"`
	UseCachedMetrics  types.Bool                  `tfsdk:"use_cached_metrics"`
}

type KEDAAuthenticationRefModel struct {
	Name types.String `tfsdk:"name"`
	Kind types.String `tfsdk:"kind"`
}

type KEDAFallbackModel struct {
	FailureThreshold types.Int32  `tfsdk:"failure_threshold"`
	Replicas         types.Int32  `tfsdk:"replicas"`
	Behavior         types.String `tfsdk:"behavior"`
}

type KEDAAdvancedModel struct {
	RestoreToOriginalReplicaCount types.Bool   `tfsdk:"restore_to_original_replica_count"`
	AdvancedBehaviorJson          types.String `tfsdk:"advanced_behavior_json"`
}

type ContainerRuleModel struct {
	ContainerName types.String                  `tfsdk:"container_name"`
	CpuRule       *ContainerResourceConfigModel `tfsdk:"cpu_rule"`
	MemoryRule    *ContainerResourceConfigModel `tfsdk:"memory_rule"`
	GpuRule       *ContainerResourceConfigModel `tfsdk:"gpu_rule"`
}

type ContainerResourceConfigModel struct {
	Enabled                 types.Bool    `tfsdk:"enabled"`
	MinRequest              types.Int64   `tfsdk:"min_request"`
	MaxRequest              types.Int64   `tfsdk:"max_request"`
	LimitMultiplier         types.Float32 `tfsdk:"limit_multiplier"`
	LimitsAdjustmentEnabled types.Bool    `tfsdk:"limits_adjustment_enabled"`
	TargetPercentile        types.Float32 `tfsdk:"target_percentile"`
	LimitsRemovalEnabled    types.Bool    `tfsdk:"limits_removal_enabled"`
	RequestUseRss           types.Bool    `tfsdk:"request_use_rss"`
	LimitUseRss             types.Bool    `tfsdk:"limit_use_rss"`
}

func emergencyResponseAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"oom_enabled": schema.BoolAttribute{
			Description: "React to OOM kills by increasing memory",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"oom_memory_multiplier": schema.Float32Attribute{
			Description: "Multiplier applied to memory on OOM",
			Optional:    true,
		},
		"oom_max_reactions": schema.Int32Attribute{
			Description: "Maximum number of OOM reactions before giving up",
			Optional:    true,
			Computed:    true,
		},
		"oom_cooldown_seconds": schema.Int32Attribute{
			Description: "Seconds to wait between OOM reactions",
			Optional:    true,
			Computed:    true,
		},
		"cpu_throttling_enabled": schema.BoolAttribute{
			Description: "React to CPU throttling by increasing CPU request",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"cpu_throttling_threshold": schema.Float32Attribute{
			Description: "Throttle ratio threshold that triggers a reaction (0-1)",
			Optional:    true,
		},
		"cpu_throttling_multiplier": schema.Float32Attribute{
			Description: "Multiplier applied to CPU request on throttle reaction",
			Optional:    true,
		},
	}
}

func hpaScalingRulesAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"stabilization_window_seconds": schema.Int32Attribute{
			Description: "Seconds to wait before acting on a scaling signal to avoid flapping",
			Optional:    true,
		},
		"select_policy": schema.StringAttribute{
			Description: "Which policy wins when multiple match. One of: 'Max', 'Min', 'Disabled'",
			Optional:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("Max", "Min", "Disabled"),
			},
		},
		"policies": schema.ListNestedAttribute{
			Description: "List of scaling step policies",
			Optional:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Policy type. One of: 'Pods', 'Percent'",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("Pods", "Percent"),
						},
					},
					"value": schema.Int32Attribute{
						Description: "Policy value (pods count or percent)",
						Required:    true,
					},
					"period_seconds": schema.Int32Attribute{
						Description: "Period over which the policy applies in seconds",
						Required:    true,
					},
				},
			},
		},
	}
}

func (r *WorkloadRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_rule"
}

func (r *WorkloadRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resourceRuleConfigAttributes := func() map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Enable this resource axis rule",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"min_request": schema.Int64Attribute{
				Description: "Minimum resource request (millicores for CPU, bytes for memory/GPU)",
				Optional:    true,
			},
			"max_request": schema.Int64Attribute{
				Description: "Maximum resource request (millicores for CPU, bytes for memory/GPU)",
				Optional:    true,
			},
			"limit_multiplier": schema.Float32Attribute{
				Description: "Multiplier applied to the request to derive the resource limit",
				Optional:    true,
			},
			"limits_adjustment_enabled": schema.BoolAttribute{
				Description: "Whether to also adjust resource limits alongside requests",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"target_percentile": schema.Float32Attribute{
				Description: "Percentile of usage data used as the recommendation target (0-1)",
				Optional:    true,
			},
			"max_scale_up_percent": schema.Float32Attribute{
				Description: "Maximum percentage increase allowed in a single cycle",
				Optional:    true,
			},
			"max_scale_down_percent": schema.Float32Attribute{
				Description: "Maximum percentage decrease allowed in a single cycle",
				Optional:    true,
			},
			"limits_removal_enabled": schema.BoolAttribute{
				Description: "Actively remove limits from workloads",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"initial_request": schema.Int64Attribute{
				Description: "Baseline request the floor/ceiling percents are computed against",
				Optional:    true,
			},
			"floor_percent": schema.Int64Attribute{
				Description: "Floor for the request as a percent of initial_request (1-100)",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 100)},
			},
			"ceiling_percent": schema.Int64Attribute{
				Description: "Ceiling for the request as a percent of initial_request (1-1000)",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 1000)},
			},
			"initial_limit": schema.Int64Attribute{
				Description: "Baseline limit the limit floor/ceiling percents are computed against",
				Optional:    true,
			},
			"limit_floor_percent": schema.Int64Attribute{
				Description: "Floor for the limit as a percent of initial_limit (1-100)",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 100)},
			},
			"limit_ceiling_percent": schema.Int64Attribute{
				Description: "Ceiling for the limit as a percent of initial_limit (1-1000)",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(1, 1000)},
			},
			"request_use_rss": schema.BoolAttribute{
				Description: "Memory only: size the request from RSS instead of working set",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"limit_use_rss": schema.BoolAttribute{
				Description: "Memory only: derive the limit from an RSS-based recommendation",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		}
	}

	containerResourceConfigAttributes := func() map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Enable this resource axis rule",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"min_request": schema.Int64Attribute{
				Description: "Minimum resource request",
				Optional:    true,
			},
			"max_request": schema.Int64Attribute{
				Description: "Maximum resource request",
				Optional:    true,
			},
			"limit_multiplier": schema.Float32Attribute{
				Description: "Multiplier applied to the request to derive the resource limit",
				Optional:    true,
			},
			"limits_adjustment_enabled": schema.BoolAttribute{
				Description: "Whether to also adjust resource limits alongside requests",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"target_percentile": schema.Float32Attribute{
				Description: "Percentile of usage data used as the recommendation target (0-1)",
				Optional:    true,
			},
			"limits_removal_enabled": schema.BoolAttribute{
				Description: "Actively remove limits from workloads",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"request_use_rss": schema.BoolAttribute{
				Description: "Memory only: size the request from RSS instead of working set",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"limit_use_rss": schema.BoolAttribute{
				Description: "Memory only: derive the limit from an RSS-based recommendation",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DevZero workload rule that configures vertical and horizontal scaling for a specific Kubernetes workload.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the workload rule",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description: "ID of the cluster this rule targets",
				Required:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Kubernetes namespace of the workload",
				Required:    true,
			},
			"kind": schema.StringAttribute{
				Description: "Kubernetes workload kind",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("Deployment", "StatefulSet", "DaemonSet", "CronJob", "Job"),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the Kubernetes workload",
				Required:    true,
			},
			"auto_generate": schema.BoolAttribute{
				Description: "When true the engine generates all rule fields automatically; manual field overrides are ignored",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"cpu_rule": schema.SingleNestedAttribute{
				Description: "CPU vertical scaling rule configuration",
				Optional:    true,
				Attributes:  resourceRuleConfigAttributes(),
			},
			"memory_rule": schema.SingleNestedAttribute{
				Description: "Memory vertical scaling rule configuration",
				Optional:    true,
				Attributes:  resourceRuleConfigAttributes(),
			},
			"gpu_rule": schema.SingleNestedAttribute{
				Description: "GPU vertical scaling rule configuration",
				Optional:    true,
				Attributes:  resourceRuleConfigAttributes(),
			},
			"hpa_rule": schema.SingleNestedAttribute{
				Description: "Horizontal (replica) scaling rule configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Enable horizontal (replica) scaling",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"min_replicas": schema.Int32Attribute{
						Description: "Minimum number of replicas",
						Optional:    true,
					},
					"max_replicas": schema.Int32Attribute{
						Description: "Maximum number of replicas",
						Optional:    true,
					},
					"max_replica_change_percent": schema.Float32Attribute{
						Description: "Maximum percentage change in replica count per cycle",
						Optional:    true,
					},
					"scale_down_cooldown_seconds": schema.Int32Attribute{
						Description: "Seconds to wait between scale-down events",
						Optional:    true,
					},
					"composite_formula": schema.StringAttribute{
						Description: "Formula combining multiple metric weights into a single scaling signal. Example: '0.6*cpu + 0.4*memory'",
						Optional:    true,
					},
					"metrics": schema.ListNestedAttribute{
						Description: "HPA metric triggers (CPU, Memory, Network*, or external such as prometheus/kafka). Replaces the removed target_utilization/primary_metric fields.",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Metric source type. Example: 'CPU', 'Memory', 'prometheus'",
									Required:    true,
								},
								"target_utilization": schema.StringAttribute{
									Description: "Target utilization as a decimal string. Example: '0.70'",
									Optional:    true,
								},
								"target_value": schema.StringAttribute{
									Description: "Absolute target value as a string. Example: '50000000'",
									Optional:    true,
								},
								"weight": schema.StringAttribute{
									Description: "Weight for composite formula scaling (0-1 decimal string). Example: '0.5'",
									Optional:    true,
								},
								"metadata": schema.MapAttribute{
									Description: "Free-form key-value metadata for external scalers (e.g. serverAddress, query for Prometheus).",
									Optional:    true,
									ElementType: types.StringType,
								},
								"server_address": schema.StringAttribute{
									Description: "Prometheus server URL. Packed into metadata by the service layer.",
									Optional:    true,
								},
								"query": schema.StringAttribute{
									Description: "PromQL query string. Packed into metadata by the service layer.",
									Optional:    true,
								},
								"connector_id": schema.StringAttribute{
									Description: "HPA connector ID to source this metric's connection settings from",
									Optional:    true,
								},
							},
						},
					},
					"fallback": schema.SingleNestedAttribute{
						Description: "Replica fallback configuration when metrics are unavailable",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"replicas": schema.Int32Attribute{
								Description: "Number of replicas to fall back to when metrics are unavailable",
								Required:    true,
							},
							"behavior": schema.StringAttribute{
								Description: "Fallback strategy. One of: 'static', 'currentReplicas', 'currentReplicasIfHigher', 'currentReplicasIfLower'",
								Optional:    true,
							},
							"failure_threshold": schema.Int32Attribute{
								Description: "Number of consecutive metric failures before activating fallback",
								Optional:    true,
							},
						},
					},
					"behavior": schema.SingleNestedAttribute{
						Description: "Fine-grained scale-up and scale-down behavior policies",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"scale_up": schema.SingleNestedAttribute{
								Description: "Scale-up behavior rules",
								Optional:    true,
								Attributes:  hpaScalingRulesAttributes(),
							},
							"scale_down": schema.SingleNestedAttribute{
								Description: "Scale-down behavior rules",
								Optional:    true,
								Attributes:  hpaScalingRulesAttributes(),
							},
						},
					},
				},
			},
			"emergency_response": schema.SingleNestedAttribute{
				Description: "Emergency response configuration for OOM and CPU throttle events",
				Optional:    true,
				Attributes:  emergencyResponseAttributes(),
			},
			"jvm_heap_rule": schema.SingleNestedAttribute{
				Description: "JVM heap optimization overrides for this rule",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Enable JVM heap optimization",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"target_percentile": schema.Float32Attribute{
						Description: "Percentile of heap usage data used as the recommendation target (0-1)",
						Optional:    true,
					},
					"headroom_multiplier": schema.Float32Attribute{
						Description: "Multiplier applied to the target heap usage to derive the recommended max heap",
						Optional:    true,
					},
					"non_heap_overhead_percent": schema.Float32Attribute{
						Description: "Non-heap memory overhead as a percentage of heap size",
						Optional:    true,
					},
					"non_heap_overhead_bytes": schema.Int64Attribute{
						Description: "Non-heap memory overhead in bytes, added on top of non_heap_overhead_percent",
						Optional:    true,
					},
					"min_heap_bytes": schema.Int64Attribute{
						Description: "Minimum recommended max heap size in bytes",
						Optional:    true,
					},
					"max_heap_bytes": schema.Int64Attribute{
						Description: "Maximum recommended max heap size in bytes",
						Optional:    true,
					},
					"prefer_container_support": schema.BoolAttribute{
						Description: "Prefer the JVM's own container-aware ergonomics (-XX:+UseContainerSupport) over an explicit -Xmx",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			"jvm_cpu_startup_floor_millicores": schema.Int64Attribute{
				Description:         "Per-rule override of the JVM CPU startup floor in millicores",
				MarkdownDescription: "Per-rule override of the JVM CPU startup floor in millicores. Unset inherits the policy/system default (75m); explicit `0` disables the floor for this rule. Always-on for detected JVMs, independent of `jvm_heap_rule.enabled`.",
				Optional:            true,
			},
			"keda_scaled_object": schema.SingleNestedAttribute{
				Description:         "KEDA ScaledObject template",
				MarkdownDescription: "KEDA ScaledObject template authored by the user. When set, the in-cluster operator owns the ScaledObject lifecycle (create/update/delete) instead of generating its own HPA.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"triggers": schema.ListNestedAttribute{
						Description: "KEDA scale triggers",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "KEDA scaler type. Example: 'prometheus', 'cpu', 'kafka'",
									Required:    true,
								},
								"name": schema.StringAttribute{
									Description: "Trigger name",
									Optional:    true,
								},
								"metadata": schema.MapAttribute{
									Description: "Scaler-specific metadata, as required by the chosen KEDA scaler type",
									Optional:    true,
									ElementType: types.StringType,
								},
								"metric_type": schema.StringAttribute{
									Description: "Metric target type. One of: 'Value', 'AverageValue', 'Utilization'",
									Optional:    true,
								},
								"authentication_ref": schema.SingleNestedAttribute{
									Description: "Reference to a KEDA TriggerAuthentication/ClusterTriggerAuthentication",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{
											Description: "Name of the referenced authentication resource",
											Required:    true,
										},
										"kind": schema.StringAttribute{
											Description: "Kind of the referenced authentication resource. One of: 'TriggerAuthentication', 'ClusterTriggerAuthentication'",
											Optional:    true,
										},
									},
								},
								"use_cached_metrics": schema.BoolAttribute{
									Description: "Use KEDA's cached metrics for this trigger",
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(false),
								},
							},
						},
					},
					"min_replica_count": schema.Int32Attribute{
						Description: "Minimum number of replicas",
						Optional:    true,
					},
					"max_replica_count": schema.Int32Attribute{
						Description: "Maximum number of replicas",
						Optional:    true,
					},
					"idle_replica_count": schema.Int32Attribute{
						Description: "Number of replicas to scale down to when idle",
						Optional:    true,
					},
					"polling_interval": schema.Int32Attribute{
						Description: "Seconds between checks of the trigger sources",
						Optional:    true,
					},
					"cooldown_period": schema.Int32Attribute{
						Description: "Seconds to wait after the last trigger reported active before scaling down to idle/min replicas",
						Optional:    true,
					},
					"initial_cooldown_period": schema.Int32Attribute{
						Description: "Cooldown period applied only on initial ScaledObject creation",
						Optional:    true,
					},
					"fallback": schema.SingleNestedAttribute{
						Description: "Replica fallback configuration when the scaler's metrics are unavailable",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"failure_threshold": schema.Int32Attribute{
								Description: "Number of consecutive metric failures before activating fallback",
								Optional:    true,
								Computed:    true,
								Default:     int32default.StaticInt32(0),
							},
							"replicas": schema.Int32Attribute{
								Description: "Number of replicas to fall back to when metrics are unavailable",
								Optional:    true,
								Computed:    true,
								Default:     int32default.StaticInt32(0),
							},
							"behavior": schema.StringAttribute{
								Description: "Fallback strategy",
								Optional:    true,
							},
						},
					},
					"advanced": schema.SingleNestedAttribute{
						Description: "Advanced KEDA ScaledObject settings",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"restore_to_original_replica_count": schema.BoolAttribute{
								Description: "Restore the original replica count when the ScaledObject is deleted",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"advanced_behavior_json": schema.StringAttribute{
								Description:         "Opaque JSON-encoded Kubernetes HorizontalPodAutoscalerBehavior",
								MarkdownDescription: "Opaque JSON-encoded Kubernetes `HorizontalPodAutoscalerBehavior`, carried through verbatim so this provider never has to re-model Kubernetes autoscaling types.",
								Optional:            true,
							},
						},
					},
				},
			},
			"action_triggers": schema.ListAttribute{
				Description: "When to apply recommendations. Valid values: 'on_detection', 'on_schedule'",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.NoNullValues(),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("on_schedule", "on_detection")),
				},
			},
			"startup_period_seconds": schema.Int64Attribute{
				Description: "Seconds after workload start to exclude from usage data",
				Optional:    true,
			},
			"cron_schedule": schema.StringAttribute{
				Description: "Cron expression for scheduled application (5-field UTC)",
				Optional:    true,
			},
			"cooldown_minutes": schema.Int32Attribute{
				Description: "Minimum minutes between consecutive recommendation applications",
				Optional:    true,
			},
			"detection_triggers": schema.ListAttribute{
				Description: "Events that trigger a recommendation. Valid values: 'pod_creation', 'pod_update'",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.NoNullValues(),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("pod_creation", "pod_update", "pod_evict")),
				},
			},
			"scheduler_plugins": schema.ListAttribute{
				Description: "Kubernetes scheduler plugins to activate",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.NoNullValues(),
					listvalidator.UniqueValues(),
				},
			},
			"defragmentation_schedule": schema.StringAttribute{
				Description: "Cron expression for node defragmentation",
				Optional:    true,
			},
			"live_migration_enabled": schema.BoolAttribute{
				Description: "Allow live pod migration when applying recommendations",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"use_in_place_vertical_scaling": schema.BoolAttribute{
				Description: "Use in-place pod vertical scaling instead of pod restarts",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"allow_in_place_memory_limit_decrease": schema.BoolAttribute{
				Description:         "Allow an in-place resize to lower a container's memory limit",
				MarkdownDescription: "Opt-in: allow an in-place resize to lower a container's memory limit. Only consulted when `use_in_place_vertical_scaling` is true; a decrease still additionally requires a cluster new enough to accept one. Default false because shrinking a live container's memory limit can OOM-kill it.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"disabled": schema.BoolAttribute{
				Description:         "Create the rule in a disabled state",
				MarkdownDescription: "Whether the rule is disabled. A disabled rule exists but is not evaluated. Changing this after creation uses the ToggleWorkloadRuleDisabled API (the upsert API rejects `disabled` on update).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"lookback_period_seconds": schema.Int32Attribute{
				Description:         "Per-rule override of the metrics lookback window (seconds)",
				MarkdownDescription: "Per-rule override of the metrics lookback window in seconds. Unset inherits the team default (7 days). Minimum 3600 (1h), maximum 2592000 (30d); higher tiers may be capped server-side.",
				Optional:            true,
				Validators:          []validator.Int32{int32validator.Between(3600, 2592000)},
			},
			"containers": schema.ListNestedAttribute{
				Description: "Per-container resource rule configurations. When empty, workload-level rules apply to all containers.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"container_name": schema.StringAttribute{
							Description: "Name of the container this config applies to",
							Required:    true,
						},
						"cpu_rule": schema.SingleNestedAttribute{
							Description: "CPU resource rule for this container",
							Optional:    true,
							Attributes:  containerResourceConfigAttributes(),
						},
						"memory_rule": schema.SingleNestedAttribute{
							Description: "Memory resource rule for this container",
							Optional:    true,
							Attributes:  containerResourceConfigAttributes(),
						},
						"gpu_rule": schema.SingleNestedAttribute{
							Description: "GPU resource rule for this container",
							Optional:    true,
							Attributes:  containerResourceConfigAttributes(),
						},
					},
				},
			},
		},
	}
}

func (r *WorkloadRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkloadRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkloadRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	upsertReq := data.toProto(ctx, &resp.Diagnostics, r.client.TeamId, true)
	if resp.Diagnostics.HasError() {
		return
	}

	upsertResp, err := r.client.RecommendationClient.UpsertManualWorkloadRule(ctx, connect.NewRequest(upsertReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workload rule, got error: %s", err))
		return
	}
	if upsertResp.Msg.Rule == nil {
		resp.Diagnostics.AddError("Client Error", "Workload rule not created")
		return
	}

	plan := data
	data.fromProto(upsertResp.Msg.Rule)
	data.preserveNullsFrom(&plan)

	tflog.Trace(ctx, "created a workload rule resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkloadRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getRuleResp, err := r.client.RecommendationClient.GetWorkloadRuleByID(ctx, connect.NewRequest(&apiv1.GetWorkloadRuleByIDRequest{
		TeamId: r.client.TeamId,
		RuleId: data.Id.ValueString(),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workload rule, got error: %s", err))
		return
	}
	if getRuleResp.Msg.Rule == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	prior := data
	data.fromProto(getRuleResp.Msg.Rule)
	if !prior.Name.IsNull() {
		// prior.Name is only null right after import (ImportStatePassthroughID
		// only sets id), where there is no real prior config to preserve nulls
		// from and fromProto's result should be trusted as-is.
		data.preserveNullsFrom(&prior)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkloadRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkloadRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	upsertReq := data.toProto(ctx, &resp.Diagnostics, r.client.TeamId, false)
	if resp.Diagnostics.HasError() {
		return
	}

	upsertResp, err := r.client.RecommendationClient.UpsertManualWorkloadRule(ctx, connect.NewRequest(upsertReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update workload rule, got error: %s", err))
		return
	}
	if upsertResp.Msg.Rule == nil {
		resp.Diagnostics.AddError("Client Error", "Workload rule not updated")
		return
	}

	// disabled cannot be sent through the upsert on update; use the toggle RPC.
	if !data.Disabled.IsNull() && !data.Disabled.IsUnknown() && data.Disabled.ValueBool() != state.Disabled.ValueBool() {
		toggleResp, err := r.client.RecommendationClient.ToggleWorkloadRuleDisabled(ctx, connect.NewRequest(&apiv1.ToggleWorkloadRuleDisabledRequest{
			TeamId:   r.client.TeamId,
			RuleId:   upsertResp.Msg.Rule.RuleId,
			Disabled: data.Disabled.ValueBool(),
		}))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to toggle workload rule disabled state, got error: %s", err))
			return
		}
		if toggleResp.Msg.Rule != nil {
			upsertResp.Msg.Rule.Disabled = toggleResp.Msg.Rule.Disabled
		} else {
			upsertResp.Msg.Rule.Disabled = data.Disabled.ValueBool()
		}
	}

	plan := data
	data.fromProto(upsertResp.Msg.Rule)
	data.preserveNullsFrom(&plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkloadRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RecommendationClient.DeleteWorkloadRule(ctx, connect.NewRequest(&apiv1.DeleteWorkloadRuleRequest{
		TeamId: r.client.TeamId,
		RuleId: data.Id.ValueString(),
	}))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workload rule, got error: %s", err))
		return
	}
}

func (r *WorkloadRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------- toProto / fromProto ----------

// toProto builds the upsert request. includeDisabled must be true only on
// Create: the server rejects fields.disabled when updating an existing rule
// (ToggleWorkloadRuleDisabled is the update path, handled in Update).
func (m *WorkloadRuleResourceModel) toProto(ctx context.Context, diags *diag.Diagnostics, teamId string, includeDisabled bool) *apiv1.UpsertManualWorkloadRuleRequest {
	source := apiv1.WorkloadRuleSource_WORKLOAD_RULE_SOURCE_TERRAFORM_MANUAL
	if m.AutoGenerate.ValueBool() {
		source = apiv1.WorkloadRuleSource_WORKLOAD_RULE_SOURCE_TERRAFORM_AUTO
	}

	req := &apiv1.UpsertManualWorkloadRuleRequest{
		TeamId:       teamId,
		ClusterId:    m.ClusterId.ValueString(),
		Namespace:    m.Namespace.ValueString(),
		Kind:         m.Kind.ValueString(),
		Name:         m.Name.ValueString(),
		AutoGenerate: m.AutoGenerate.ValueBool(),
		Source:       source,
	}

	if m.AutoGenerate.ValueBool() {
		return req
	}

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
		diags.AddError("Client Error", fmt.Sprintf("Unable to convert action triggers: %s", err))
		return nil
	}

	detectionTriggers, err := getElementList(ctx, m.DetectionTriggers.Elements(), func(ctx context.Context, value string) (apiv1.WorkloadDetectionTrigger, error) {
		switch value {
		case "pod_creation":
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION, nil
		case "pod_update":
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_UPDATE, nil
		case "pod_evict":
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_EVICT, nil
		default:
			return apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_UNSPECIFIED, fmt.Errorf("invalid detection trigger: %s", value)
		}
	})
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to convert detection triggers: %s", err))
		return nil
	}

	schedulerPlugins, err := getStringList(ctx, m.SchedulerPlugins.Elements())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to convert scheduler plugins: %s", err))
		return nil
	}

	fields := &apiv1.ManualRuleFields{
		CpuRule:                         m.CpuRule.toProto(),
		MemoryRule:                      m.MemoryRule.toProto(),
		GpuRule:                         m.GpuRule.toProto(),
		HpaRule:                         m.HpaRule.toProto(),
		EmergencyResponse:               m.EmergencyResponse.toProto(),
		ActionTriggers:                  actionTriggers,
		DetectionTriggers:               detectionTriggers,
		SchedulerPlugins:                schedulerPlugins,
		LiveMigrationEnabled:            m.LiveMigrationEnabled.ValueBool(),
		UseInPlaceVerticalScaling:       m.UseInPlaceVerticalScaling.ValueBool(),
		AllowInPlaceMemoryLimitDecrease: m.AllowInPlaceMemoryLimitDecrease.ValueBool(),
		Containers:                      containerRuleModelsToProto(m.Containers),
		JvmHeapRule:                     m.JvmHeapRule.toProto(),
		JvmCpuStartupFloorMillicores:    m.JvmCpuStartupFloorMillicores.ValueInt64Pointer(),
		KedaScaledObject:                m.KedaScaledObject.toProto(),
	}

	if !m.StartupPeriodSeconds.IsNull() && !m.StartupPeriodSeconds.IsUnknown() {
		v := m.StartupPeriodSeconds.ValueInt64()
		fields.StartupPeriodSeconds = &v
	}
	if !m.CronSchedule.IsNull() && !m.CronSchedule.IsUnknown() {
		v := m.CronSchedule.ValueString()
		fields.CronSchedule = &v
	}
	if !m.CooldownMinutes.IsNull() && !m.CooldownMinutes.IsUnknown() {
		v := m.CooldownMinutes.ValueInt32()
		fields.CooldownMinutes = &v
	}
	if !m.DefragmentationSchedule.IsNull() && !m.DefragmentationSchedule.IsUnknown() {
		v := m.DefragmentationSchedule.ValueString()
		fields.DefragmentationSchedule = &v
	}
	if !m.LookbackPeriodSeconds.IsNull() && !m.LookbackPeriodSeconds.IsUnknown() {
		v := m.LookbackPeriodSeconds.ValueInt32()
		fields.LookbackPeriodSeconds = &v
	}
	if includeDisabled && !m.Disabled.IsNull() && !m.Disabled.IsUnknown() {
		v := m.Disabled.ValueBool()
		fields.Disabled = &v
	}

	req.Fields = fields
	return req
}

func (m *WorkloadRuleResourceModel) preserveNullsFrom(plan *WorkloadRuleResourceModel) {
	if plan.CpuRule == nil {
		m.CpuRule = nil
	}
	if plan.MemoryRule == nil {
		m.MemoryRule = nil
	}
	if plan.GpuRule == nil {
		m.GpuRule = nil
	}
	if plan.HpaRule == nil {
		m.HpaRule = nil
	}
	if plan.EmergencyResponse == nil {
		m.EmergencyResponse = nil
	}
	if plan.JvmHeapRule == nil {
		m.JvmHeapRule = nil
	}
	if plan.KedaScaledObject == nil {
		m.KedaScaledObject = nil
	}
	if plan.JvmCpuStartupFloorMillicores.IsNull() {
		m.JvmCpuStartupFloorMillicores = types.Int64Null()
	}
	if plan.ActionTriggers.IsNull() {
		m.ActionTriggers = types.ListNull(types.StringType)
	}
	if plan.DetectionTriggers.IsNull() {
		m.DetectionTriggers = types.ListNull(types.StringType)
	}
	if plan.SchedulerPlugins.IsNull() {
		m.SchedulerPlugins = types.ListNull(types.StringType)
	}
	if plan.CooldownMinutes.IsNull() {
		m.CooldownMinutes = types.Int32Null()
	}
	if plan.StartupPeriodSeconds.IsNull() {
		m.StartupPeriodSeconds = types.Int64Null()
	}
	if plan.CronSchedule.IsNull() {
		m.CronSchedule = types.StringNull()
	}
	if plan.DefragmentationSchedule.IsNull() {
		m.DefragmentationSchedule = types.StringNull()
	}
	if plan.Containers == nil {
		m.Containers = nil
	}
}

func (m *WorkloadRuleResourceModel) fromProto(r *apiv1.WorkloadRule) {
	m.Id = types.StringValue(r.RuleId)
	m.ClusterId = types.StringValue(r.ClusterId)
	m.Namespace = types.StringValue(r.Namespace)
	m.Kind = types.StringValue(r.Kind)
	m.Name = types.StringValue(r.Name)

	switch r.CurrentSource {
	case "auto_optimization", "terraform_auto", "pulumi_auto":
		m.AutoGenerate = types.BoolValue(true)
	default:
		m.AutoGenerate = types.BoolValue(false)
	}

	m.Disabled = types.BoolValue(r.Disabled)
	m.LookbackPeriodSeconds = types.Int32PointerValue(r.LookbackPeriodSeconds)

	m.CpuRule = resourceRuleConfigFromProto(r.CpuRule)
	m.MemoryRule = resourceRuleConfigFromProto(r.MemoryRule)
	m.GpuRule = resourceRuleConfigFromProto(r.GpuRule)
	m.HpaRule = hpaRuleConfigFromProto(r.HpaRule)
	m.EmergencyResponse = emergencyResponseFromProto(r.EmergencyResponse)
	m.AllowInPlaceMemoryLimitDecrease = types.BoolValue(r.AllowInPlaceMemoryLimitDecrease)
	m.JvmHeapRule = jvmHeapRuleFromProto(r.JvmHeapRule)
	m.JvmCpuStartupFloorMillicores = types.Int64PointerValue(r.JvmCpuStartupFloorMillicores)
	m.KedaScaledObject = kedaScaledObjectFromProto(r.KedaScaledObject)

	actionTriggers := make([]attr.Value, 0)
	for _, at := range r.ActionTriggers {
		switch at {
		case apiv1.ActionTrigger_ACTION_TRIGGER_ON_SCHEDULE:
			actionTriggers = append(actionTriggers, types.StringValue("on_schedule"))
		case apiv1.ActionTrigger_ACTION_TRIGGER_ON_DETECTION:
			actionTriggers = append(actionTriggers, types.StringValue("on_detection"))
		}
	}
	m.ActionTriggers = types.ListValueMust(types.StringType, actionTriggers)

	if r.StartupPeriodSeconds != nil {
		m.StartupPeriodSeconds = types.Int64Value(*r.StartupPeriodSeconds)
	} else {
		m.StartupPeriodSeconds = types.Int64Null()
	}

	if r.CronSchedule != nil {
		m.CronSchedule = types.StringValue(*r.CronSchedule)
	} else {
		m.CronSchedule = types.StringNull()
	}

	if r.CooldownMinutes != nil {
		m.CooldownMinutes = types.Int32Value(*r.CooldownMinutes)
	} else {
		m.CooldownMinutes = types.Int32Null()
	}

	detectionTriggers := make([]attr.Value, 0)
	for _, dt := range r.DetectionTriggers {
		switch dt {
		case apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION:
			detectionTriggers = append(detectionTriggers, types.StringValue("pod_creation"))
		case apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_UPDATE:
			detectionTriggers = append(detectionTriggers, types.StringValue("pod_update"))
		case apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_EVICT:
			detectionTriggers = append(detectionTriggers, types.StringValue("pod_evict"))
		}
	}
	m.DetectionTriggers = types.ListValueMust(types.StringType, detectionTriggers)

	var schedulerPlugins []attr.Value
	for _, sp := range r.SchedulerPlugins {
		schedulerPlugins = append(schedulerPlugins, types.StringValue(sp))
	}
	m.SchedulerPlugins = types.ListValueMust(types.StringType, schedulerPlugins)

	if r.DefragmentationSchedule != nil {
		m.DefragmentationSchedule = types.StringValue(*r.DefragmentationSchedule)
	} else {
		m.DefragmentationSchedule = types.StringNull()
	}

	m.LiveMigrationEnabled = types.BoolValue(r.LiveMigrationEnabled)
	m.UseInPlaceVerticalScaling = types.BoolValue(r.UseInPlaceVerticalScaling)
	m.Containers = containerRuleModelsFromProto(r.Containers)
}

// ---------- ResourceRuleConfig ----------

func (m *ResourceRuleConfigModel) toProto() *apiv1.ResourceRuleConfig {
	if m == nil {
		return nil
	}
	p := &apiv1.ResourceRuleConfig{
		Enabled:                 m.Enabled.ValueBool(),
		LimitsAdjustmentEnabled: m.LimitsAdjustmentEnabled.ValueBool(),
		LimitsRemovalEnabled:    m.LimitsRemovalEnabled.ValueBool(),
	}
	if !m.MinRequest.IsNull() && !m.MinRequest.IsUnknown() {
		v := m.MinRequest.ValueInt64()
		p.MinRequest = &v
	}
	if !m.MaxRequest.IsNull() && !m.MaxRequest.IsUnknown() {
		v := m.MaxRequest.ValueInt64()
		p.MaxRequest = &v
	}
	if !m.LimitMultiplier.IsNull() && !m.LimitMultiplier.IsUnknown() {
		v := m.LimitMultiplier.ValueFloat32()
		p.LimitMultiplier = &v
	}
	if !m.TargetPercentile.IsNull() && !m.TargetPercentile.IsUnknown() {
		v := m.TargetPercentile.ValueFloat32()
		p.TargetPercentile = &v
	}
	if !m.MaxScaleUpPercent.IsNull() && !m.MaxScaleUpPercent.IsUnknown() {
		v := m.MaxScaleUpPercent.ValueFloat32()
		p.MaxScaleUpPercent = &v
	}
	if !m.MaxScaleDownPercent.IsNull() && !m.MaxScaleDownPercent.IsUnknown() {
		v := m.MaxScaleDownPercent.ValueFloat32()
		p.MaxScaleDownPercent = &v
	}
	p.InitialRequest = m.InitialRequest.ValueInt64Pointer()
	p.FloorPercent = m.FloorPercent.ValueInt64Pointer()
	p.CeilingPercent = m.CeilingPercent.ValueInt64Pointer()
	p.InitialLimit = m.InitialLimit.ValueInt64Pointer()
	p.LimitFloorPercent = m.LimitFloorPercent.ValueInt64Pointer()
	p.LimitCeilingPercent = m.LimitCeilingPercent.ValueInt64Pointer()
	p.RequestUseRss = m.RequestUseRss.ValueBoolPointer()
	p.LimitUseRss = m.LimitUseRss.ValueBoolPointer()
	return p
}

func resourceRuleConfigFromProto(p *apiv1.ResourceRuleConfig) *ResourceRuleConfigModel {
	if p == nil {
		return nil
	}
	m := &ResourceRuleConfigModel{
		Enabled:                 types.BoolValue(p.Enabled),
		LimitsAdjustmentEnabled: types.BoolValue(p.LimitsAdjustmentEnabled),
		LimitsRemovalEnabled:    types.BoolValue(p.LimitsRemovalEnabled),
		MinRequest:              types.Int64Null(),
		MaxRequest:              types.Int64Null(),
		LimitMultiplier:         types.Float32Null(),
		TargetPercentile:        types.Float32Null(),
		MaxScaleUpPercent:       types.Float32Null(),
		MaxScaleDownPercent:     types.Float32Null(),
		InitialRequest:          types.Int64PointerValue(p.InitialRequest),
		FloorPercent:            types.Int64PointerValue(p.FloorPercent),
		CeilingPercent:          types.Int64PointerValue(p.CeilingPercent),
		InitialLimit:            types.Int64PointerValue(p.InitialLimit),
		LimitFloorPercent:       types.Int64PointerValue(p.LimitFloorPercent),
		LimitCeilingPercent:     types.Int64PointerValue(p.LimitCeilingPercent),
		RequestUseRss:           types.BoolValue(p.RequestUseRss != nil && *p.RequestUseRss),
		LimitUseRss:             types.BoolValue(p.LimitUseRss != nil && *p.LimitUseRss),
	}
	if p.MinRequest != nil {
		m.MinRequest = types.Int64Value(*p.MinRequest)
	}
	if p.MaxRequest != nil {
		m.MaxRequest = types.Int64Value(*p.MaxRequest)
	}
	if p.LimitMultiplier != nil {
		m.LimitMultiplier = types.Float32Value(*p.LimitMultiplier)
	}
	if p.TargetPercentile != nil {
		m.TargetPercentile = types.Float32Value(*p.TargetPercentile)
	}
	if p.MaxScaleUpPercent != nil {
		m.MaxScaleUpPercent = types.Float32Value(*p.MaxScaleUpPercent)
	}
	if p.MaxScaleDownPercent != nil {
		m.MaxScaleDownPercent = types.Float32Value(*p.MaxScaleDownPercent)
	}
	return m
}

// ---------- HPARuleConfig ----------

func (m *HPARuleConfigModel) toProto() *apiv1.HPARuleConfig {
	if m == nil {
		return nil
	}
	p := &apiv1.HPARuleConfig{
		Enabled: m.Enabled.ValueBool(),
	}
	if !m.MinReplicas.IsNull() && !m.MinReplicas.IsUnknown() {
		v := m.MinReplicas.ValueInt32()
		p.MinReplicas = &v
	}
	if !m.MaxReplicas.IsNull() && !m.MaxReplicas.IsUnknown() {
		v := m.MaxReplicas.ValueInt32()
		p.MaxReplicas = &v
	}
	if !m.MaxReplicaChangePercent.IsNull() && !m.MaxReplicaChangePercent.IsUnknown() {
		v := m.MaxReplicaChangePercent.ValueFloat32()
		p.MaxReplicaChangePercent = &v
	}
	if !m.ScaleDownCooldownSeconds.IsNull() && !m.ScaleDownCooldownSeconds.IsUnknown() {
		v := m.ScaleDownCooldownSeconds.ValueInt32()
		p.ScaleDownCooldownSeconds = &v
	}
	if !m.CompositeFormula.IsNull() && !m.CompositeFormula.IsUnknown() {
		v := m.CompositeFormula.ValueString()
		p.CompositeFormula = &v
	}
	if len(m.Metrics) > 0 {
		p.Metrics = hpaMetricTriggersToProto(m.Metrics)
	}
	if m.Behavior != nil {
		p.Behavior = hpaBehaviorToProto(m.Behavior)
	}
	if m.Fallback != nil {
		p.Fallback = hpaFallbackToProto(m.Fallback)
	}
	return p
}

func hpaRuleConfigFromProto(p *apiv1.HPARuleConfig) *HPARuleConfigModel {
	if p == nil {
		return nil
	}
	m := &HPARuleConfigModel{
		Enabled:                  types.BoolValue(p.Enabled),
		MinReplicas:              types.Int32Null(),
		MaxReplicas:              types.Int32Null(),
		MaxReplicaChangePercent:  types.Float32Null(),
		ScaleDownCooldownSeconds: types.Int32Null(),
		CompositeFormula:         types.StringNull(),
	}
	if p.MinReplicas != nil {
		m.MinReplicas = types.Int32Value(*p.MinReplicas)
	}
	if p.MaxReplicas != nil {
		m.MaxReplicas = types.Int32Value(*p.MaxReplicas)
	}
	if p.MaxReplicaChangePercent != nil {
		m.MaxReplicaChangePercent = types.Float32Value(*p.MaxReplicaChangePercent)
	}
	if p.ScaleDownCooldownSeconds != nil {
		m.ScaleDownCooldownSeconds = types.Int32Value(*p.ScaleDownCooldownSeconds)
	}
	if p.CompositeFormula != nil {
		m.CompositeFormula = types.StringValue(*p.CompositeFormula)
	}
	if len(p.Metrics) > 0 {
		m.Metrics = hpaMetricTriggersFromProto(p.Metrics)
	}
	if p.Behavior != nil {
		m.Behavior = hpaBehaviorFromProto(p.Behavior)
	}
	if p.Fallback != nil {
		m.Fallback = hpaFallbackFromProto(p.Fallback)
	}
	return m
}

// ---------- EmergencyResponse ----------

func (m *EmergencyResponseModel) toProto() *apiv1.EmergencyResponseConfig {
	if m == nil {
		return nil
	}
	p := &apiv1.EmergencyResponseConfig{
		OomEnabled:           m.OomEnabled.ValueBool(),
		CpuThrottlingEnabled: m.CpuThrottlingEnabled.ValueBool(),
	}
	if !m.OomMemoryMultiplier.IsNull() && !m.OomMemoryMultiplier.IsUnknown() {
		p.OomMemoryMultiplier = m.OomMemoryMultiplier.ValueFloat32()
	}
	if !m.OomMaxReactions.IsNull() && !m.OomMaxReactions.IsUnknown() {
		p.OomMaxReactions = m.OomMaxReactions.ValueInt32()
	}
	if !m.OomCooldownSeconds.IsNull() && !m.OomCooldownSeconds.IsUnknown() {
		p.OomCooldownSeconds = m.OomCooldownSeconds.ValueInt32()
	}
	if !m.CpuThrottlingThreshold.IsNull() && !m.CpuThrottlingThreshold.IsUnknown() {
		p.CpuThrottlingThreshold = m.CpuThrottlingThreshold.ValueFloat32()
	}
	if !m.CpuThrottlingMultiplier.IsNull() && !m.CpuThrottlingMultiplier.IsUnknown() {
		p.CpuThrottlingMultiplier = m.CpuThrottlingMultiplier.ValueFloat32()
	}
	return p
}

func emergencyResponseFromProto(p *apiv1.EmergencyResponseConfig) *EmergencyResponseModel {
	if p == nil {
		return nil
	}
	return &EmergencyResponseModel{
		OomEnabled:              types.BoolValue(p.OomEnabled),
		OomMemoryMultiplier:     float32OrNull(p.OomMemoryMultiplier),
		OomMaxReactions:         types.Int32Value(p.OomMaxReactions),
		OomCooldownSeconds:      types.Int32Value(p.OomCooldownSeconds),
		CpuThrottlingEnabled:    types.BoolValue(p.CpuThrottlingEnabled),
		CpuThrottlingThreshold:  float32OrNull(p.CpuThrottlingThreshold),
		CpuThrottlingMultiplier: float32OrNull(p.CpuThrottlingMultiplier),
	}
}

// ---------- JVMHeapRule ----------

func (m *JVMHeapRuleModel) toProto() *apiv1.JVMHeapRuleConfig {
	if m == nil {
		return nil
	}
	return &apiv1.JVMHeapRuleConfig{
		Enabled:                m.Enabled.ValueBool(),
		TargetPercentile:       m.TargetPercentile.ValueFloat32Pointer(),
		HeadroomMultiplier:     m.HeadroomMultiplier.ValueFloat32Pointer(),
		NonHeapOverheadPercent: m.NonHeapOverheadPercent.ValueFloat32Pointer(),
		NonHeapOverheadBytes:   m.NonHeapOverheadBytes.ValueInt64Pointer(),
		MinHeapBytes:           m.MinHeapBytes.ValueInt64Pointer(),
		MaxHeapBytes:           m.MaxHeapBytes.ValueInt64Pointer(),
		PreferContainerSupport: m.PreferContainerSupport.ValueBool(),
	}
}

func jvmHeapRuleFromProto(p *apiv1.JVMHeapRuleConfig) *JVMHeapRuleModel {
	if p == nil {
		return nil
	}
	return &JVMHeapRuleModel{
		Enabled:                types.BoolValue(p.Enabled),
		TargetPercentile:       types.Float32PointerValue(p.TargetPercentile),
		HeadroomMultiplier:     types.Float32PointerValue(p.HeadroomMultiplier),
		NonHeapOverheadPercent: types.Float32PointerValue(p.NonHeapOverheadPercent),
		NonHeapOverheadBytes:   types.Int64PointerValue(p.NonHeapOverheadBytes),
		MinHeapBytes:           types.Int64PointerValue(p.MinHeapBytes),
		MaxHeapBytes:           types.Int64PointerValue(p.MaxHeapBytes),
		PreferContainerSupport: types.BoolValue(p.PreferContainerSupport),
	}
}

// ---------- KEDAScaledObject ----------

func (m *KEDAScaledObjectModel) toProto() *apiv1.KEDAScaledObjectTemplate {
	if m == nil {
		return nil
	}
	p := &apiv1.KEDAScaledObjectTemplate{
		Triggers:              kedaTriggersToProto(m.Triggers),
		MinReplicaCount:       m.MinReplicaCount.ValueInt32Pointer(),
		MaxReplicaCount:       m.MaxReplicaCount.ValueInt32Pointer(),
		IdleReplicaCount:      m.IdleReplicaCount.ValueInt32Pointer(),
		PollingInterval:       m.PollingInterval.ValueInt32Pointer(),
		CooldownPeriod:        m.CooldownPeriod.ValueInt32Pointer(),
		InitialCooldownPeriod: m.InitialCooldownPeriod.ValueInt32Pointer(),
	}
	if m.Fallback != nil {
		p.Fallback = &apiv1.KEDAFallback{
			FailureThreshold: m.Fallback.FailureThreshold.ValueInt32(),
			Replicas:         m.Fallback.Replicas.ValueInt32(),
			Behavior:         m.Fallback.Behavior.ValueString(),
		}
	}
	if m.Advanced != nil {
		p.Advanced = &apiv1.KEDAAdvanced{
			RestoreToOriginalReplicaCount: m.Advanced.RestoreToOriginalReplicaCount.ValueBool(),
			AdvancedBehaviorJson:          m.Advanced.AdvancedBehaviorJson.ValueString(),
		}
	}
	return p
}

func kedaScaledObjectFromProto(p *apiv1.KEDAScaledObjectTemplate) *KEDAScaledObjectModel {
	if p == nil {
		return nil
	}
	m := &KEDAScaledObjectModel{
		Triggers:              kedaTriggersFromProto(p.Triggers),
		MinReplicaCount:       types.Int32PointerValue(p.MinReplicaCount),
		MaxReplicaCount:       types.Int32PointerValue(p.MaxReplicaCount),
		IdleReplicaCount:      types.Int32PointerValue(p.IdleReplicaCount),
		PollingInterval:       types.Int32PointerValue(p.PollingInterval),
		CooldownPeriod:        types.Int32PointerValue(p.CooldownPeriod),
		InitialCooldownPeriod: types.Int32PointerValue(p.InitialCooldownPeriod),
	}
	if p.Fallback != nil {
		m.Fallback = &KEDAFallbackModel{
			FailureThreshold: types.Int32Value(p.Fallback.FailureThreshold),
			Replicas:         types.Int32Value(p.Fallback.Replicas),
			Behavior:         stringValue(p.Fallback.Behavior),
		}
	}
	if p.Advanced != nil {
		m.Advanced = &KEDAAdvancedModel{
			RestoreToOriginalReplicaCount: types.BoolValue(p.Advanced.RestoreToOriginalReplicaCount),
			AdvancedBehaviorJson:          stringValue(p.Advanced.AdvancedBehaviorJson),
		}
	}
	return m
}

func kedaTriggersToProto(ts []KEDATriggerModel) []*apiv1.KEDATrigger {
	if len(ts) == 0 {
		return nil
	}
	result := make([]*apiv1.KEDATrigger, len(ts))
	for i, t := range ts {
		kt := &apiv1.KEDATrigger{
			Type:             t.Type.ValueString(),
			Name:             t.Name.ValueString(),
			MetricType:       t.MetricType.ValueString(),
			UseCachedMetrics: t.UseCachedMetrics.ValueBool(),
		}
		if !t.Metadata.IsNull() && !t.Metadata.IsUnknown() {
			meta := make(map[string]string, len(t.Metadata.Elements()))
			for k, v := range t.Metadata.Elements() {
				if sv, ok := v.(types.String); ok {
					meta[k] = sv.ValueString()
				}
			}
			kt.Metadata = meta
		}
		if t.AuthenticationRef != nil {
			kt.AuthenticationRef = &apiv1.KEDAAuthenticationRef{
				Name: t.AuthenticationRef.Name.ValueString(),
				Kind: t.AuthenticationRef.Kind.ValueString(),
			}
		}
		result[i] = kt
	}
	return result
}

func kedaTriggersFromProto(ps []*apiv1.KEDATrigger) []KEDATriggerModel {
	if len(ps) == 0 {
		return nil
	}
	result := make([]KEDATriggerModel, 0, len(ps))
	for _, p := range ps {
		if p == nil {
			continue
		}
		m := KEDATriggerModel{
			Type:             types.StringValue(p.Type),
			Name:             stringValue(p.Name),
			Metadata:         stringMapOrNull(p.Metadata),
			MetricType:       stringValue(p.MetricType),
			UseCachedMetrics: types.BoolValue(p.UseCachedMetrics),
		}
		if p.AuthenticationRef != nil {
			m.AuthenticationRef = &KEDAAuthenticationRefModel{
				Name: types.StringValue(p.AuthenticationRef.Name),
				Kind: stringValue(p.AuthenticationRef.Kind),
			}
		}
		result = append(result, m)
	}
	return result
}

// ---------- Containers ----------

func containerRuleModelsToProto(cs []ContainerRuleModel) []*apiv1.ContainerResourceRuleConfig {
	if len(cs) == 0 {
		return nil
	}
	result := make([]*apiv1.ContainerResourceRuleConfig, len(cs))
	for i, c := range cs {
		result[i] = &apiv1.ContainerResourceRuleConfig{
			ContainerName: c.ContainerName.ValueString(),
			CpuRule:       c.CpuRule.toProto(),
			MemoryRule:    c.MemoryRule.toProto(),
			GpuRule:       c.GpuRule.toProto(),
		}
	}
	return result
}

func containerRuleModelsFromProto(ps []*apiv1.ContainerResourceRuleConfig) []ContainerRuleModel {
	if len(ps) == 0 {
		return nil
	}
	result := make([]ContainerRuleModel, len(ps))
	for i, p := range ps {
		result[i] = ContainerRuleModel{
			ContainerName: types.StringValue(p.ContainerName),
			CpuRule:       containerResourceConfigFromProto(p.CpuRule),
			MemoryRule:    containerResourceConfigFromProto(p.MemoryRule),
			GpuRule:       containerResourceConfigFromProto(p.GpuRule),
		}
	}
	return result
}

func (m *ContainerResourceConfigModel) toProto() *apiv1.ContainerResourceConfig {
	if m == nil {
		return nil
	}
	p := &apiv1.ContainerResourceConfig{
		Enabled:                 m.Enabled.ValueBool(),
		LimitsAdjustmentEnabled: m.LimitsAdjustmentEnabled.ValueBool(),
		LimitsRemovalEnabled:    m.LimitsRemovalEnabled.ValueBool(),
	}
	if !m.MinRequest.IsNull() && !m.MinRequest.IsUnknown() {
		v := m.MinRequest.ValueInt64()
		p.MinRequest = &v
	}
	if !m.MaxRequest.IsNull() && !m.MaxRequest.IsUnknown() {
		v := m.MaxRequest.ValueInt64()
		p.MaxRequest = &v
	}
	if !m.LimitMultiplier.IsNull() && !m.LimitMultiplier.IsUnknown() {
		v := m.LimitMultiplier.ValueFloat32()
		p.LimitMultiplier = &v
	}
	if !m.TargetPercentile.IsNull() && !m.TargetPercentile.IsUnknown() {
		v := m.TargetPercentile.ValueFloat32()
		p.TargetPercentile = &v
	}
	p.RequestUseRss = m.RequestUseRss.ValueBoolPointer()
	p.LimitUseRss = m.LimitUseRss.ValueBoolPointer()
	return p
}

func containerResourceConfigFromProto(p *apiv1.ContainerResourceConfig) *ContainerResourceConfigModel {
	if p == nil {
		return nil
	}
	m := &ContainerResourceConfigModel{
		Enabled:                 types.BoolValue(p.Enabled),
		LimitsAdjustmentEnabled: types.BoolValue(p.LimitsAdjustmentEnabled),
		LimitsRemovalEnabled:    types.BoolValue(p.LimitsRemovalEnabled),
		MinRequest:              types.Int64Null(),
		MaxRequest:              types.Int64Null(),
		LimitMultiplier:         types.Float32Null(),
		TargetPercentile:        types.Float32Null(),
		RequestUseRss:           types.BoolValue(p.RequestUseRss != nil && *p.RequestUseRss),
		LimitUseRss:             types.BoolValue(p.LimitUseRss != nil && *p.LimitUseRss),
	}
	if p.MinRequest != nil {
		m.MinRequest = types.Int64Value(*p.MinRequest)
	}
	if p.MaxRequest != nil {
		m.MaxRequest = types.Int64Value(*p.MaxRequest)
	}
	if p.LimitMultiplier != nil {
		m.LimitMultiplier = types.Float32Value(*p.LimitMultiplier)
	}
	if p.TargetPercentile != nil {
		m.TargetPercentile = types.Float32Value(*p.TargetPercentile)
	}
	return m
}

// ---------- HPA helpers ----------

func hpaMetricTriggersToProto(ms []HPAMetricTriggerModel) []*apiv1.HPAMetricTrigger {
	result := make([]*apiv1.HPAMetricTrigger, len(ms))
	for i, m := range ms {
		t := &apiv1.HPAMetricTrigger{Type: m.Type.ValueString()}
		if !m.TargetUtilization.IsNull() && !m.TargetUtilization.IsUnknown() {
			v := m.TargetUtilization.ValueString()
			t.TargetUtilization = &v
		}
		if !m.TargetValue.IsNull() && !m.TargetValue.IsUnknown() {
			v := m.TargetValue.ValueString()
			t.TargetValue = &v
		}
		if !m.Weight.IsNull() && !m.Weight.IsUnknown() {
			v := m.Weight.ValueString()
			t.Weight = &v
		}
		if !m.Metadata.IsNull() && !m.Metadata.IsUnknown() {
			meta := make(map[string]string)
			for k, v := range m.Metadata.Elements() {
				if sv, ok := v.(types.String); ok {
					meta[k] = sv.ValueString()
				}
			}
			t.Metadata = meta
		}
		if !m.ServerAddress.IsNull() && !m.ServerAddress.IsUnknown() {
			v := m.ServerAddress.ValueString()
			t.ServerAddress = &v
		}
		if !m.Query.IsNull() && !m.Query.IsUnknown() {
			v := m.Query.ValueString()
			t.Query = &v
		}
		if !m.ConnectorId.IsNull() && !m.ConnectorId.IsUnknown() {
			v := m.ConnectorId.ValueString()
			t.ConnectorId = &v
		}
		result[i] = t
	}
	return result
}

func hpaMetricTriggersFromProto(ps []*apiv1.HPAMetricTrigger) []HPAMetricTriggerModel {
	result := make([]HPAMetricTriggerModel, 0, len(ps))
	for _, p := range ps {
		if p == nil {
			continue
		}
		m := HPAMetricTriggerModel{
			Type:              types.StringValue(p.Type),
			TargetUtilization: types.StringNull(),
			TargetValue:       types.StringNull(),
			Weight:            types.StringNull(),
			Metadata:          types.MapNull(types.StringType),
			ServerAddress:     types.StringNull(),
			Query:             types.StringNull(),
			ConnectorId:       types.StringNull(),
		}
		if p.TargetUtilization != nil {
			m.TargetUtilization = types.StringValue(*p.TargetUtilization)
		}
		if p.TargetValue != nil {
			m.TargetValue = types.StringValue(*p.TargetValue)
		}
		if p.Weight != nil {
			m.Weight = types.StringValue(*p.Weight)
		}
		// The backend folds server_address/query into metadata["serverAddress"/"query"]
		// on write and re-derives the dedicated fields on read. Strip the folded keys
		// so metadata reflects only what the user configured.
		metadata := make(map[string]string, len(p.Metadata))
		for k, v := range p.Metadata {
			if k == "serverAddress" || k == "query" {
				continue
			}
			metadata[k] = v
		}
		if len(metadata) > 0 {
			m.Metadata = types.MapValueMust(types.StringType, fromStringMap(metadata))
		}
		if p.ServerAddress != nil && *p.ServerAddress != "" {
			m.ServerAddress = types.StringValue(*p.ServerAddress)
		}
		if p.Query != nil && *p.Query != "" {
			m.Query = types.StringValue(*p.Query)
		}
		if p.ConnectorId != nil && *p.ConnectorId != "" {
			m.ConnectorId = types.StringValue(*p.ConnectorId)
		}
		result = append(result, m)
	}
	return result
}

func hpaFallbackToProto(f *HPAFallbackModel) *apiv1.HPAFallback {
	if f == nil {
		return nil
	}
	p := &apiv1.HPAFallback{
		Replicas: f.Replicas.ValueInt32(),
	}
	if !f.Behavior.IsNull() && !f.Behavior.IsUnknown() {
		p.Behavior = f.Behavior.ValueString()
	}
	if !f.FailureThreshold.IsNull() && !f.FailureThreshold.IsUnknown() {
		p.FailureThreshold = f.FailureThreshold.ValueInt32()
	}
	return p
}

func hpaFallbackFromProto(p *apiv1.HPAFallback) *HPAFallbackModel {
	if p == nil {
		return nil
	}
	return &HPAFallbackModel{
		Replicas:         types.Int32Value(p.Replicas),
		Behavior:         stringValue(p.Behavior),
		FailureThreshold: int32OrNull(p.FailureThreshold),
	}
}

func hpaBehaviorToProto(b *HPABehaviorModel) *apiv1.HPABehavior {
	if b == nil {
		return nil
	}
	return &apiv1.HPABehavior{
		ScaleUp:   hpaScalingRulesToProto(b.ScaleUp),
		ScaleDown: hpaScalingRulesToProto(b.ScaleDown),
	}
}

func hpaBehaviorFromProto(p *apiv1.HPABehavior) *HPABehaviorModel {
	if p == nil {
		return nil
	}
	return &HPABehaviorModel{
		ScaleUp:   hpaScalingRulesFromProto(p.ScaleUp),
		ScaleDown: hpaScalingRulesFromProto(p.ScaleDown),
	}
}

func hpaScalingRulesToProto(r *HPAScalingRulesModel) *apiv1.HPAScalingRules {
	if r == nil {
		return nil
	}
	p := &apiv1.HPAScalingRules{
		StabilizationWindowSeconds: r.StabilizationWindowSeconds.ValueInt32(),
		SelectPolicy:               r.SelectPolicy.ValueString(),
	}
	for _, pol := range r.Policies {
		p.Policies = append(p.Policies, &apiv1.HPAScalingPolicy{
			Type:          pol.Type.ValueString(),
			Value:         pol.Value.ValueInt32(),
			PeriodSeconds: pol.PeriodSeconds.ValueInt32(),
		})
	}
	return p
}

func hpaScalingRulesFromProto(p *apiv1.HPAScalingRules) *HPAScalingRulesModel {
	if p == nil {
		return nil
	}
	m := &HPAScalingRulesModel{
		StabilizationWindowSeconds: int32OrNull(p.StabilizationWindowSeconds),
		SelectPolicy:               stringValue(p.SelectPolicy),
	}
	for _, pol := range p.Policies {
		if pol == nil {
			continue
		}
		m.Policies = append(m.Policies, HPAScalingPolicyModel{
			Type:          types.StringValue(pol.Type),
			Value:         types.Int32Value(pol.Value),
			PeriodSeconds: types.Int32Value(pol.PeriodSeconds),
		})
	}
	return m
}

// float32OrNull returns null for the proto zero value so omitted optional
// attributes round-trip as null instead of producing an
// "inconsistent result after apply" error.
func float32OrNull(v float32) types.Float32 {
	if v == 0 {
		return types.Float32Null()
	}
	return types.Float32Value(v)
}

func int32OrNull(v int32) types.Int32 {
	if v == 0 {
		return types.Int32Null()
	}
	return types.Int32Value(v)
}
