package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &NodePolicyResource{}
	_ resource.ResourceWithConfigure   = &NodePolicyResource{}
	_ resource.ResourceWithImportState = &NodePolicyResource{}
)

func NewNodePolicyResource() resource.Resource {
	return &NodePolicyResource{}
}

// NodePolicyResource defines the resource implementation.
type NodePolicyResource struct {
	client *ClientSet
}

// NodePolicyResourceModel describes the resource data model.
type NodePolicyResourceModel struct {
	Id                     types.String      `tfsdk:"id"`
	Name                   types.String      `tfsdk:"name"`
	Description            types.String      `tfsdk:"description"`
	Weight                 types.Int32       `tfsdk:"weight"`
	InstanceCategories     *LabelSelector    `tfsdk:"instance_categories"`
	InstanceFamilies       *LabelSelector    `tfsdk:"instance_families"`
	InstanceCpus           *LabelSelector    `tfsdk:"instance_cpus"`
	InstanceHypervisors    *LabelSelector    `tfsdk:"instance_hypervisors"`
	InstanceGenerations    *LabelSelector    `tfsdk:"instance_generations"`
	InstanceSizes          *LabelSelector    `tfsdk:"instance_sizes"`
	InstanceCategoriesTip  types.String      `tfsdk:"instance_categories_tip"`
	InstanceFamiliesTip    types.String      `tfsdk:"instance_families_tip"`
	InstanceCpusTip        types.String      `tfsdk:"instance_cpus_tip"`
	InstanceHypervisorsTip types.String      `tfsdk:"instance_hypervisors_tip"`
	InstanceGenerationsTip types.String      `tfsdk:"instance_generations_tip"`
	InstanceSizesTip       types.String      `tfsdk:"instance_sizes_tip"`
	Zones                  *LabelSelector    `tfsdk:"zones"`
	Architectures          *LabelSelector    `tfsdk:"architectures"`
	CapacityTypes          *LabelSelector    `tfsdk:"capacity_types"`
	OperatingSystems       *LabelSelector    `tfsdk:"operating_systems"`
	ZonesTip               types.String      `tfsdk:"zones_tip"`
	ArchitecturesTip       types.String      `tfsdk:"architectures_tip"`
	CapacityTypeTip        types.String      `tfsdk:"capacity_type_tip"`
	OperatingSystemsTip    types.String      `tfsdk:"operating_systems_tip"`
	Labels                 types.Map         `tfsdk:"labels"`
	Taints                 types.List        `tfsdk:"taints"` // List of Taint objects
	Disruption             *DisruptionPolicy `tfsdk:"disruption"`
	Limits                 *ResourceLimits   `tfsdk:"limits"`
	TaintsTip              types.String      `tfsdk:"taints_tip"`
	DisruptionsTip         types.String      `tfsdk:"disruptions_tip"`
	LimitsTip              types.String      `tfsdk:"limits_tip"`
	MasterOverrideRoleName types.String      `tfsdk:"master_override_role_name"`
	NodePoolName           types.String      `tfsdk:"node_pool_name"`
	NodeClassName          types.String      `tfsdk:"node_class_name"`
	Aws                    *AWSNodeClass     `tfsdk:"aws"`
	Azure                  *AzureNodeClass   `tfsdk:"azure"`
	Raw                    types.List        `tfsdk:"raw"` // List of RawKarpenterSpec objects
}

// Taint defines Kubernetes taints.
type Taint struct {
	Key    types.String `tfsdk:"key"`
	Value  types.String `tfsdk:"value"`
	Effect types.String `tfsdk:"effect"`
}

// DisruptionPolicy defines node disruption policy.
type DisruptionPolicy struct {
	ConsolidateAfter              types.String `tfsdk:"consolidate_after"`
	ConsolidationPolicy           types.String `tfsdk:"consolidation_policy"`
	ExpireAfter                   types.String `tfsdk:"expire_after"`
	TtlSecondsAfterEmpty          types.Int32  `tfsdk:"ttl_seconds_after_empty"`
	TerminationGracePeriodSeconds types.Int32  `tfsdk:"termination_grace_period_seconds"`
	Budgets                       types.List   `tfsdk:"budgets"` // List of DisruptionBudget
}

// DisruptionBudget defines disruption budget constraints.
type DisruptionBudget struct {
	Reasons  types.List   `tfsdk:"reasons"` // List of strings
	Nodes    types.String `tfsdk:"nodes"`
	Schedule types.String `tfsdk:"schedule"`
	Duration types.String `tfsdk:"duration"`
}

// ResourceLimits defines resource limits for nodes.
type ResourceLimits struct {
	Cpu    types.String `tfsdk:"cpu"`
	Memory types.String `tfsdk:"memory"`
}

// MetadataOptions defines EC2 instance metadata service options.
type MetadataOptions struct {
	HttpEndpoint            types.String `tfsdk:"http_endpoint"`
	HttpProtocolIpv6        types.String `tfsdk:"http_protocol_ipv6"`
	HttpPutResponseHopLimit types.Int64  `tfsdk:"http_put_response_hop_limit"`
	HttpTokens              types.String `tfsdk:"http_tokens"`
}

// AWSNodeClass defines AWS-specific node configuration.
type AWSNodeClass struct {
	SubnetSelectorTerms        types.List       `tfsdk:"subnet_selector_terms"`
	SecurityGroupSelectorTerms types.List       `tfsdk:"security_group_selector_terms"`
	AmiSelectorTerms           types.List       `tfsdk:"ami_selector_terms"`
	AmiFamily                  types.String     `tfsdk:"ami_family"`
	UserData                   types.String     `tfsdk:"user_data"`
	Role                       types.String     `tfsdk:"role"`
	InstanceProfile            types.String     `tfsdk:"instance_profile"`
	Tags                       types.Map        `tfsdk:"tags"`
	BlockDeviceMappings        types.List       `tfsdk:"block_device_mappings"`
	InstanceStorePolicy        types.String     `tfsdk:"instance_store_policy"`
	DetailedMonitoring         types.Bool       `tfsdk:"detailed_monitoring"`
	AssociatePublicIpAddress   types.Bool       `tfsdk:"associate_public_ip_address"`
	MetadataOptions            *MetadataOptions `tfsdk:"metadata_options"`
}

// AzureNodeClass defines Azure-specific node configuration.
type AzureNodeClass struct {
	VnetSubnetId types.String `tfsdk:"vnet_subnet_id"`
	OsDiskSizeGb types.Int32  `tfsdk:"os_disk_size_gb"`
	ImageFamily  types.String `tfsdk:"image_family"`
	FipsMode     types.String `tfsdk:"fips_mode"`
	Tags         types.Map    `tfsdk:"tags"`
	MaxPods      types.Int32  `tfsdk:"max_pods"`
}

// RawKarpenterSpec defines raw Karpenter YAML specs.
type RawKarpenterSpec struct {
	NodepoolYaml  types.String `tfsdk:"nodepool_yaml"`
	NodeclassYaml types.String `tfsdk:"nodeclass_yaml"`
}

func (r *NodePolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_policy"
}

func (r *NodePolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages DevZero node policies for Kubernetes cluster node provisioning and optimization using Karpenter.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Unique identifier of the node policy",
				MarkdownDescription: "Unique identifier of the node policy. Managed by the provider.",
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
			"weight": schema.Int32Attribute{
				Description:         "Priority weight for this node policy",
				MarkdownDescription: "Priority weight for this node policy. Higher weights are preferred when multiple policies match. Default: 10 (medium priority).",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(10),
			},
			// Instance selector fields with LabelSelector
			"instance_categories":  labelSelectorAttribute("Instance categories selector (e.g., D for Azure, m for AWS)"),
			"instance_families":    labelSelectorAttribute("Instance families selector (e.g., c5, m5d, r4)"),
			"instance_cpus":        labelSelectorAttribute("Instance CPU count selector (e.g., 4, 8, 16)"),
			"instance_hypervisors": labelSelectorAttribute("Instance hypervisors selector"),
			"instance_generations": labelSelectorAttribute("Instance generations selector (e.g., 4 for Azure, 5 for AWS)"),
			"instance_sizes":       labelSelectorAttribute("Instance sizes selector (e.g., Standard_D4s for Azure, large for AWS)"),
			// Tooltip fields for instance selectors
			"instance_categories_tip":  tooltipAttribute("Tooltip for instance categories"),
			"instance_families_tip":    tooltipAttribute("Tooltip for instance families"),
			"instance_cpus_tip":        tooltipAttribute("Tooltip for instance CPUs"),
			"instance_hypervisors_tip": tooltipAttribute("Tooltip for instance hypervisors"),
			"instance_generations_tip": tooltipAttribute("Tooltip for instance generations"),
			"instance_sizes_tip":       tooltipAttribute("Tooltip for instance sizes"),
			// Additional selectors
			"zones":             labelSelectorAttribute("Availability zones selector"),
			"architectures":     labelSelectorAttribute("CPU architectures selector (e.g., amd64, arm64)"),
			"capacity_types":    labelSelectorAttribute("Capacity types selector (e.g., spot, on-demand, reserved)"),
			"operating_systems": labelSelectorAttribute("Operating systems selector (e.g., linux, windows)"),
			// Tooltips for additional selectors
			"zones_tip":             tooltipAttribute("Tooltip for zones"),
			"architectures_tip":     tooltipAttribute("Tooltip for architectures"),
			"capacity_type_tip":     tooltipAttribute("Tooltip for capacity types"),
			"operating_systems_tip": tooltipAttribute("Tooltip for operating systems"),
			// Node configuration
			"labels": schema.MapAttribute{
				Description:         "Kubernetes labels to apply to nodes",
				MarkdownDescription: "Map of Kubernetes labels to apply to nodes provisioned with this policy.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"taints": schema.ListNestedAttribute{
				Description:         "Kubernetes taints to apply to nodes",
				MarkdownDescription: "List of Kubernetes taints to apply to nodes provisioned with this policy.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "Taint key",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "Taint value",
							Required:    true,
						},
						"effect": schema.StringAttribute{
							Description:         "Taint effect (NoSchedule, PreferNoSchedule, NoExecute)",
							MarkdownDescription: "Taint effect. Valid values: `NoSchedule`, `PreferNoSchedule`, `NoExecute`.",
							Required:            true,
						},
					},
				},
			},
			"disruption": schema.SingleNestedAttribute{
				Description:         "Node disruption policy configuration",
				MarkdownDescription: "Configuration for node disruption policies including consolidation and expiration settings.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"consolidate_after": schema.StringAttribute{
						Description:         "Duration after which to consolidate nodes",
						MarkdownDescription: "Duration string (e.g., '5m', '1h') after which nodes can be consolidated. Default: '15m' (balance between cost optimization and stability).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("15m"),
					},
					"consolidation_policy": schema.StringAttribute{
						Description:         "Consolidation policy (WhenEmpty, WhenEmptyOrUnderutilized)",
						MarkdownDescription: "Consolidation policy. Valid values: `WhenEmpty`, `WhenEmptyOrUnderutilized`. Default: 'WhenEmptyOrUnderutilized' (best for cost optimization).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("WhenEmptyOrUnderutilized"),
					},
					"expire_after": schema.StringAttribute{
						Description:         "Duration after which nodes expire",
						MarkdownDescription: "Duration string (e.g., '720h') after which nodes expire and are replaced. Default: '720h' (30 days, balances security and stability).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("720h"),
					},
					"ttl_seconds_after_empty": schema.Int32Attribute{
						Description: "Seconds to wait before terminating empty nodes",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(0),
					},
					"termination_grace_period_seconds": schema.Int32Attribute{
						Description: "Grace period for node termination",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(0),
					},
					"budgets": schema.ListNestedAttribute{
						Description: "Disruption budgets",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"reasons": schema.ListAttribute{
									Description:         "Reasons for disruption (e.g., Underutilized, Empty)",
									MarkdownDescription: "List of reasons that trigger this budget. Examples: `Underutilized`, `Empty`.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"nodes": schema.StringAttribute{
									Description:         "Node limit (e.g., '10%' or '2')",
									MarkdownDescription: "Maximum nodes that can be disrupted, as percentage (e.g., '10%') or absolute number (e.g., '2').",
									Optional:            true,
								},
								"schedule": schema.StringAttribute{
									Description: "Cron schedule for when this budget applies",
									Optional:    true,
								},
								"duration": schema.StringAttribute{
									Description:         "Duration for this budget",
									MarkdownDescription: "Duration string (e.g., '1h30m') for how long this budget applies.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"limits": schema.SingleNestedAttribute{
				Description:         "Resource limits for nodes",
				MarkdownDescription: "Maximum resource limits for nodes provisioned with this policy.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"cpu": schema.StringAttribute{
						Description:         "Maximum CPU limit",
						MarkdownDescription: "Maximum CPU limit for nodes (e.g., '100', '1000').",
						Optional:            true,
					},
					"memory": schema.StringAttribute{
						Description:         "Maximum memory limit",
						MarkdownDescription: "Maximum memory limit for nodes (e.g., '512Gi', '1Ti').",
						Optional:            true,
					},
				},
			},
			// Tooltips for node configuration
			"taints_tip":      tooltipAttribute("Tooltip for taints"),
			"disruptions_tip": tooltipAttribute("Tooltip for disruptions"),
			"limits_tip":      tooltipAttribute("Tooltip for limits"),
			// Karpenter naming
			"master_override_role_name": schema.StringAttribute{
				Description: "Master override role name for Karpenter",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"node_pool_name": schema.StringAttribute{
				Description: "Node pool name",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"node_class_name": schema.StringAttribute{
				Description: "Node class name",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			// AWS provider configuration
			"aws": schema.SingleNestedAttribute{
				Description:         "AWS-specific node configuration",
				MarkdownDescription: "AWS-specific configuration for nodes provisioned with this policy.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"subnet_selector_terms": schema.ListNestedAttribute{
						Description: "Subnet selector terms",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "Subnet ID",
									Optional:    true,
								},
								"tags": schema.MapAttribute{
									Description: "Subnet tags selector",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
					"security_group_selector_terms": schema.ListNestedAttribute{
						Description: "Security group selector terms",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "Security group ID",
									Optional:    true,
								},
								"name": schema.StringAttribute{
									Description: "Security group name",
									Optional:    true,
								},
								"tags": schema.MapAttribute{
									Description: "Security group tags selector",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
					"ami_selector_terms": schema.ListNestedAttribute{
						Description: "AMI selector terms",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "AMI ID",
									Optional:    true,
								},
								"name": schema.StringAttribute{
									Description: "AMI name",
									Optional:    true,
								},
								"owner": schema.StringAttribute{
									Description: "AMI owner",
									Optional:    true,
								},
								"alias": schema.StringAttribute{
									Description: "AMI alias",
									Optional:    true,
								},
								"tags": schema.MapAttribute{
									Description: "AMI tags selector",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
					"ami_family": schema.StringAttribute{
						Description: "AMI family (e.g., AL2, Bottlerocket, Ubuntu)",
						Optional:    true,
					},
					"user_data": schema.StringAttribute{
						Description: "User data script for instance initialization",
						Optional:    true,
					},
					"role": schema.StringAttribute{
						Description: "IAM role name",
						Optional:    true,
					},
					"instance_profile": schema.StringAttribute{
						Description: "IAM instance profile",
						Optional:    true,
					},
					"tags": schema.MapAttribute{
						Description: "AWS tags to apply to instances",
						Optional:    true,
						ElementType: types.StringType,
					},
					"block_device_mappings": schema.ListNestedAttribute{
						Description: "Block device mappings",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"device_name": schema.StringAttribute{
									Description: "Device name (e.g., /dev/xvda)",
									Optional:    true,
								},
								"ebs": schema.SingleNestedAttribute{
									Description: "EBS volume configuration",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"volume_size": schema.StringAttribute{
											Description: "Volume size (e.g., '100Gi')",
											Optional:    true,
										},
										"volume_type": schema.StringAttribute{
											Description: "Volume type (gp2, gp3, io1, io2, sc1, st1)",
											Optional:    true,
										},
										"iops": schema.Int64Attribute{
											Description: "IOPS for io1/io2 volumes",
											Optional:    true,
										},
										"throughput": schema.Int64Attribute{
											Description: "Throughput in MiB/s for gp3 volumes",
											Optional:    true,
										},
										"kms_key_id": schema.StringAttribute{
											Description: "KMS key ID for encryption",
											Optional:    true,
										},
										"snapshot_id": schema.StringAttribute{
											Description: "Snapshot ID to create volume from",
											Optional:    true,
										},
										"delete_on_termination": schema.BoolAttribute{
											Description: "Delete volume on instance termination",
											Optional:    true,
										},
										"encrypted": schema.BoolAttribute{
											Description: "Encrypt the volume",
											Optional:    true,
										},
									},
								},
							},
						},
					},
					"instance_store_policy": schema.StringAttribute{
						Description:         "Instance store policy (RAID0)",
						MarkdownDescription: "Policy for instance store volumes. Valid value: `RAID0`.",
						Optional:            true,
					},
					"detailed_monitoring": schema.BoolAttribute{
						Description: "Enable detailed CloudWatch monitoring",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"associate_public_ip_address": schema.BoolAttribute{
						Description: "Associate public IP address with instances",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"metadata_options": schema.SingleNestedAttribute{
						Description:         "EC2 instance metadata service (IMDS) options",
						MarkdownDescription: "Configuration for EC2 instance metadata service. Defaults provide secure IMDS v2 configuration.",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"http_endpoint": schema.StringAttribute{
								Description:         "Enable or disable the HTTP metadata endpoint",
								MarkdownDescription: "Enable or disable the HTTP metadata endpoint. Valid values: `enabled`, `disabled`. Default: `enabled`.",
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("enabled"),
							},
							"http_protocol_ipv6": schema.StringAttribute{
								Description:         "Enable or disable IPv6 endpoint",
								MarkdownDescription: "Enable or disable the IPv6 endpoint for the instance metadata service. Valid values: `enabled`, `disabled`. Default: `disabled`.",
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("disabled"),
							},
							"http_put_response_hop_limit": schema.Int64Attribute{
								Description:         "Desired HTTP PUT response hop limit for instance metadata requests",
								MarkdownDescription: "The desired HTTP PUT response hop limit for instance metadata requests. Default: 2 (secure for containers).",
								Optional:            true,
								Computed:            true,
								Default:             int64default.StaticInt64(2),
							},
							"http_tokens": schema.StringAttribute{
								Description:         "Whether or not the metadata service requires session tokens (IMDSv2)",
								MarkdownDescription: "Whether the metadata service requires session tokens (IMDSv2). Valid values: `required`, `optional`. Default: `required` (enforces IMDSv2 for security).",
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("required"),
							},
						},
					},
				},
			},
			// Azure provider configuration
			"azure": schema.SingleNestedAttribute{
				Description:         "Azure-specific node configuration",
				MarkdownDescription: "Azure-specific configuration for nodes provisioned with this policy.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"vnet_subnet_id": schema.StringAttribute{
						Description: "VNet subnet ID",
						Optional:    true,
					},
					"os_disk_size_gb": schema.Int32Attribute{
						Description: "OS disk size in GB",
						Optional:    true,
					},
					"image_family": schema.StringAttribute{
						Description:         "Image family (Ubuntu, Ubuntu2204, Ubuntu2404, AzureLinux)",
						MarkdownDescription: "Azure image family. Valid values: `Ubuntu`, `Ubuntu2204`, `Ubuntu2404`, `AzureLinux`.",
						Optional:            true,
					},
					"fips_mode": schema.StringAttribute{
						Description:         "FIPS mode (FIPS, Disabled)",
						MarkdownDescription: "FIPS 140-2 mode. Valid values: `FIPS`, `Disabled`.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						Description: "Azure tags to apply to resources",
						Optional:    true,
						ElementType: types.StringType,
					},
					"max_pods": schema.Int32Attribute{
						Description: "Maximum number of pods per node",
						Optional:    true,
					},
				},
			},
			// Raw Karpenter specs
			"raw": schema.ListNestedAttribute{
				Description:         "Raw Karpenter YAML specifications",
				MarkdownDescription: "Raw Karpenter NodePool and NodeClass YAML specifications for advanced use cases.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"nodepool_yaml": schema.StringAttribute{
							Description: "Raw NodePool YAML",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
						"nodeclass_yaml": schema.StringAttribute{
							Description: "Raw NodeClass YAML",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
					},
				},
			},
		},
	}
}

// Helper function to create label selector attributes.
func labelSelectorAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"match_labels": schema.MapAttribute{
				Description: "Map of label key-value pairs to match",
				Optional:    true,
				ElementType: types.StringType,
			},
			"match_expressions": schema.ListNestedAttribute{
				Description: "List of label selector requirements",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "Label key",
							Required:    true,
						},
						"operator": schema.StringAttribute{
							Description:         "Operator (In, NotIn, Exists, DoesNotExist)",
							MarkdownDescription: "Operator for matching. Valid values: `In`, `NotIn`, `Exists`, `DoesNotExist`.",
							Required:            true,
						},
						"values": schema.ListAttribute{
							Description: "List of values for In/NotIn operators",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

// Helper function to create tooltip attributes.
func tooltipAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description,
		Optional:    true,
	}
}

func (r *NodePolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NodePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NodePolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := data.toProto(ctx, &resp.Diagnostics, r.client.TeamId)
	if resp.Diagnostics.HasError() {
		return
	}

	createNodePoliciesReq := &apiv1.CreateNodePoliciesRequest{
		TeamId:   r.client.TeamId,
		Policies: []*apiv1.NodePolicy{policy}, // Wrap single policy in array
	}

	createNodePoliciesResp, err := r.client.RecommendationClient.CreateNodePolicies(ctx, connect.NewRequest(createNodePoliciesReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create node policy, got error: %s", err))
		return
	}
	if len(createNodePoliciesResp.Msg.Policies) == 0 {
		resp.Diagnostics.AddError("Client Error", "Node policy not created")
		return
	}

	data.fromProto(createNodePoliciesResp.Msg.Policies[0])

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a node policy resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NodePolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// List all policies and find the one with matching ID
	listNodePoliciesReq := &apiv1.ListNodePoliciesRequest{
		TeamId: r.client.TeamId,
	}

	listNodePoliciesResp, err := r.client.RecommendationClient.ListNodePolicies(ctx, connect.NewRequest(listNodePoliciesReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list node policies, got error: %s", err))
		return
	}

	// Find the policy with matching ID
	var foundPolicy *apiv1.NodePolicy
	for _, policy := range listNodePoliciesResp.Msg.Policies {
		if policy.Id == data.Id.ValueString() {
			foundPolicy = policy
			break
		}
	}

	if foundPolicy == nil {
		resp.Diagnostics.AddError("Client Error", "Node policy not found")
		return
	}

	data.fromProto(foundPolicy)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NodePolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateNodePolicyReq := &apiv1.UpdateNodePolicyRequest{
		TeamId: r.client.TeamId,
		Policy: data.toProto(ctx, &resp.Diagnostics, r.client.TeamId),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	updateNodePolicyResp, err := r.client.RecommendationClient.UpdateNodePolicy(ctx, connect.NewRequest(updateNodePolicyReq))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update node policy, got error: %s", err))
		return
	}

	if updateNodePolicyResp.Msg.Policy == nil {
		resp.Diagnostics.AddError("Client Error", "Node policy not updated")
		return
	}

	data.fromProto(updateNodePolicyResp.Msg.Policy)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NodePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NodePolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// No-op delete: just remove from state
	// The API doesn't provide a delete endpoint, so we just remove it from Terraform state
	// The policy will remain in the backend
	tflog.Warn(ctx, "Node policy delete is a no-op operation. The policy will remain in the backend.")
}

func (r *NodePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// toProto converts Terraform model to protobuf message.
func (m *NodePolicyResourceModel) toProto(ctx context.Context, diags *diag.Diagnostics, teamId string) *apiv1.NodePolicy {
	policy := &apiv1.NodePolicy{
		Id:          m.Id.ValueString(),
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		TeamId:      teamId,
		Weight:      m.Weight.ValueInt32(),
	}

	// Instance selectors
	if m.InstanceCategories != nil {
		selector, err := m.InstanceCategories.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert instance categories: %s", err))
			return nil
		}
		policy.InstanceCategories = selector
	}
	if m.InstanceFamilies != nil {
		selector, err := m.InstanceFamilies.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert instance families: %s", err))
			return nil
		}
		policy.InstanceFamilies = selector
	}
	if m.InstanceCpus != nil {
		selector, err := m.InstanceCpus.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert instance cpus: %s", err))
			return nil
		}
		policy.InstanceCpus = selector
	}
	if m.InstanceHypervisors != nil {
		selector, err := m.InstanceHypervisors.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert instance hypervisors: %s", err))
			return nil
		}
		policy.InstanceHypervisors = selector
	}
	if m.InstanceGenerations != nil {
		selector, err := m.InstanceGenerations.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert instance generations: %s", err))
			return nil
		}
		policy.InstanceGenerations = selector
	}
	if m.InstanceSizes != nil {
		selector, err := m.InstanceSizes.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert instance sizes: %s", err))
			return nil
		}
		policy.InstanceSizes = selector
	}

	// Tooltip fields (pointers for optional)
	if !m.InstanceCategoriesTip.IsNull() {
		val := m.InstanceCategoriesTip.ValueString()
		policy.InstanceCategoriesTip = &val
	}
	if !m.InstanceFamiliesTip.IsNull() {
		val := m.InstanceFamiliesTip.ValueString()
		policy.InstanceFamiliesTip = &val
	}
	if !m.InstanceCpusTip.IsNull() {
		val := m.InstanceCpusTip.ValueString()
		policy.InstanceCpusTip = &val
	}
	if !m.InstanceHypervisorsTip.IsNull() {
		val := m.InstanceHypervisorsTip.ValueString()
		policy.InstanceHypervisorsTip = &val
	}
	if !m.InstanceGenerationsTip.IsNull() {
		val := m.InstanceGenerationsTip.ValueString()
		policy.InstanceGenerationsTip = &val
	}
	if !m.InstanceSizesTip.IsNull() {
		val := m.InstanceSizesTip.ValueString()
		policy.InstanceSizesTip = &val
	}

	// Additional selectors
	if m.Zones != nil {
		selector, err := m.Zones.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert zones: %s", err))
			return nil
		}
		policy.Zones = selector
	}
	if m.Architectures != nil {
		selector, err := m.Architectures.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert architectures: %s", err))
			return nil
		}
		policy.Architectures = selector
	}
	if m.CapacityTypes != nil {
		selector, err := m.CapacityTypes.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert capacity types: %s", err))
			return nil
		}
		policy.CapacityTypes = selector
	}
	if m.OperatingSystems != nil {
		selector, err := m.OperatingSystems.toProto(ctx)
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert operating systems: %s", err))
			return nil
		}
		policy.OperatingSystems = selector
	}

	// Additional tooltip fields
	if !m.ZonesTip.IsNull() {
		val := m.ZonesTip.ValueString()
		policy.ZonesTip = &val
	}
	if !m.ArchitecturesTip.IsNull() {
		val := m.ArchitecturesTip.ValueString()
		policy.ArchitecturesTip = &val
	}
	if !m.CapacityTypeTip.IsNull() {
		val := m.CapacityTypeTip.ValueString()
		policy.CapacityTypeTip = &val
	}
	if !m.OperatingSystemsTip.IsNull() {
		val := m.OperatingSystemsTip.ValueString()
		policy.OperatingSystemsTip = &val
	}

	// Labels
	if !m.Labels.IsNull() {
		labels, err := getStringMap(ctx, m.Labels.Elements())
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert labels: %s", err))
			return nil
		}
		policy.Labels = labels
	}

	// Taints
	if !m.Taints.IsNull() && !m.Taints.IsUnknown() {
		taints, err := getElementList(ctx, m.Taints.Elements(), func(ctx context.Context, value Taint) (*apiv1.Taint, error) {
			return &apiv1.Taint{
				Key:    value.Key.ValueString(),
				Value:  value.Value.ValueString(),
				Effect: value.Effect.ValueString(),
			}, nil
		})
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert taints: %s", err))
			return nil
		}
		policy.Taints = taints
	}

	// Disruption policy
	if m.Disruption != nil {
		policy.Disruption = m.Disruption.toProto(ctx, diags)
	}

	// Limits
	if m.Limits != nil {
		policy.Limits = &apiv1.ResourceLimits{
			Cpu:    m.Limits.Cpu.ValueString(),
			Memory: m.Limits.Memory.ValueString(),
		}
	}

	// Tooltip fields for node config
	if !m.TaintsTip.IsNull() {
		val := m.TaintsTip.ValueString()
		policy.TaintsTip = &val
	}
	if !m.DisruptionsTip.IsNull() {
		val := m.DisruptionsTip.ValueString()
		policy.DisruptionsTip = &val
	}
	if !m.LimitsTip.IsNull() {
		val := m.LimitsTip.ValueString()
		policy.LimitsTip = &val
	}

	// Karpenter names
	policy.MasterOverrideRoleName = m.MasterOverrideRoleName.ValueString()
	policy.NodePoolName = m.NodePoolName.ValueString()
	policy.NodeClassName = m.NodeClassName.ValueString()

	// AWS configuration
	if m.Aws != nil {
		policy.Aws = m.Aws.toProto(ctx, diags)
	}

	// Azure configuration
	if m.Azure != nil {
		policy.Azure = m.Azure.toProto(ctx, diags)
	}

	// Raw Karpenter specs
	if !m.Raw.IsNull() && !m.Raw.IsUnknown() {
		rawSpecs, err := getElementList(ctx, m.Raw.Elements(), func(ctx context.Context, value RawKarpenterSpec) (*apiv1.RawKarpenterSpec, error) {
			return &apiv1.RawKarpenterSpec{
				NodepoolYaml:  value.NodepoolYaml.ValueString(),
				NodeclassYaml: value.NodeclassYaml.ValueString(),
			}, nil
		})
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert raw specs: %s", err))
			return nil
		}
		policy.Raw = rawSpecs
	}

	return policy
}

// fromProto converts protobuf message to Terraform model.
func (m *NodePolicyResourceModel) fromProto(policy *apiv1.NodePolicy) {
	m.Id = types.StringValue(policy.Id)
	m.Name = types.StringValue(policy.Name)
	m.Description = types.StringValue(policy.Description)
	m.Weight = types.Int32Value(policy.Weight)

	// Instance selectors
	if policy.InstanceCategories != nil {
		m.InstanceCategories = labelSelectorFromProto(policy.InstanceCategories)
	}
	if policy.InstanceFamilies != nil {
		m.InstanceFamilies = labelSelectorFromProto(policy.InstanceFamilies)
	}
	if policy.InstanceCpus != nil {
		m.InstanceCpus = labelSelectorFromProto(policy.InstanceCpus)
	}
	if policy.InstanceHypervisors != nil {
		m.InstanceHypervisors = labelSelectorFromProto(policy.InstanceHypervisors)
	}
	if policy.InstanceGenerations != nil {
		m.InstanceGenerations = labelSelectorFromProto(policy.InstanceGenerations)
	}
	if policy.InstanceSizes != nil {
		m.InstanceSizes = labelSelectorFromProto(policy.InstanceSizes)
	}

	// Tooltip fields
	m.InstanceCategoriesTip = stringPointerValue(policy.InstanceCategoriesTip)
	m.InstanceFamiliesTip = stringPointerValue(policy.InstanceFamiliesTip)
	m.InstanceCpusTip = stringPointerValue(policy.InstanceCpusTip)
	m.InstanceHypervisorsTip = stringPointerValue(policy.InstanceHypervisorsTip)
	m.InstanceGenerationsTip = stringPointerValue(policy.InstanceGenerationsTip)
	m.InstanceSizesTip = stringPointerValue(policy.InstanceSizesTip)

	// Additional selectors
	if policy.Zones != nil {
		m.Zones = labelSelectorFromProto(policy.Zones)
	}
	if policy.Architectures != nil {
		m.Architectures = labelSelectorFromProto(policy.Architectures)
	}
	if policy.CapacityTypes != nil {
		m.CapacityTypes = labelSelectorFromProto(policy.CapacityTypes)
	}
	if policy.OperatingSystems != nil {
		m.OperatingSystems = labelSelectorFromProto(policy.OperatingSystems)
	}

	// Additional tooltips
	m.ZonesTip = stringPointerValue(policy.ZonesTip)
	m.ArchitecturesTip = stringPointerValue(policy.ArchitecturesTip)
	m.CapacityTypeTip = stringPointerValue(policy.CapacityTypeTip)
	m.OperatingSystemsTip = stringPointerValue(policy.OperatingSystemsTip)

	// Labels
	if policy.Labels != nil {
		m.Labels = types.MapValueMust(types.StringType, fromStringMap(policy.Labels))
	} else {
		m.Labels = types.MapNull(types.StringType)
	}

	// Taints
	if len(policy.Taints) > 0 {
		taints := make([]attr.Value, 0, len(policy.Taints))
		for _, taint := range policy.Taints {
			taints = append(taints, types.ObjectValueMust(
				map[string]attr.Type{
					"key":    types.StringType,
					"value":  types.StringType,
					"effect": types.StringType,
				},
				map[string]attr.Value{
					"key":    types.StringValue(taint.Key),
					"value":  types.StringValue(taint.Value),
					"effect": types.StringValue(taint.Effect),
				},
			))
		}
		m.Taints = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":    types.StringType,
					"value":  types.StringType,
					"effect": types.StringType,
				},
			},
			taints,
		)
	} else {
		m.Taints = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":    types.StringType,
				"value":  types.StringType,
				"effect": types.StringType,
			},
		})
	}

	// Disruption policy
	if policy.Disruption != nil {
		m.Disruption = disruptionPolicyFromProto(policy.Disruption)
	}

	// Limits
	if policy.Limits != nil {
		m.Limits = &ResourceLimits{
			Cpu:    types.StringValue(policy.Limits.Cpu),
			Memory: types.StringValue(policy.Limits.Memory),
		}
	}

	// Tooltip fields for node config
	m.TaintsTip = stringPointerValue(policy.TaintsTip)
	m.DisruptionsTip = stringPointerValue(policy.DisruptionsTip)
	m.LimitsTip = stringPointerValue(policy.LimitsTip)

	// Karpenter names
	m.MasterOverrideRoleName = types.StringValue(policy.MasterOverrideRoleName)
	m.NodePoolName = types.StringValue(policy.NodePoolName)
	m.NodeClassName = types.StringValue(policy.NodeClassName)

	// AWS configuration
	if policy.Aws != nil && !isAWSSpecEmpty(policy.Aws) {
		m.Aws = awsNodeClassFromProto(policy.Aws)
	}

	// Azure configuration
	if policy.Azure != nil && !isAzureSpecEmpty(policy.Azure) {
		m.Azure = azureNodeClassFromProto(policy.Azure)
	}

	// Raw specs
	if len(policy.Raw) > 0 {
		rawSpecs := make([]attr.Value, 0, len(policy.Raw))
		for _, raw := range policy.Raw {
			rawSpecs = append(rawSpecs, types.ObjectValueMust(
				map[string]attr.Type{
					"nodepool_yaml":  types.StringType,
					"nodeclass_yaml": types.StringType,
				},
				map[string]attr.Value{
					"nodepool_yaml":  types.StringValue(raw.NodepoolYaml),
					"nodeclass_yaml": types.StringValue(raw.NodeclassYaml),
				},
			))
		}
		m.Raw = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"nodepool_yaml":  types.StringType,
					"nodeclass_yaml": types.StringType,
				},
			},
			rawSpecs,
		)
	} else {
		m.Raw = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"nodepool_yaml":  types.StringType,
				"nodeclass_yaml": types.StringType,
			},
		})
	}
}

// Helper function for converting string pointers.
func stringPointerValue(val *string) types.String {
	if val == nil {
		return types.StringNull()
	}
	return types.StringValue(*val)
}

// Helper function to convert a string to types.String, returning null for empty strings.
func stringValue(val string) types.String {
	if val == "" {
		return types.StringNull()
	}
	return types.StringValue(val)
}

// Helper function for label selector from proto.
func labelSelectorFromProto(selector *apiv1.LabelSelector) *LabelSelector {
	ls := &LabelSelector{}

	if selector.MatchLabels != nil {
		ls.MatchLabels = types.MapValueMust(types.StringType, fromStringMap(selector.MatchLabels))
	} else {
		ls.MatchLabels = types.MapNull(types.StringType)
	}

	if len(selector.MatchExpressions) > 0 {
		exprs := make([]attr.Value, 0, len(selector.MatchExpressions))
		for _, expr := range selector.MatchExpressions {
			exprs = append(exprs, types.ObjectValueMust(
				map[string]attr.Type{
					"key":      types.StringType,
					"operator": types.StringType,
					"values":   types.ListType{ElemType: types.StringType},
				},
				map[string]attr.Value{
					"key":      types.StringValue(expr.Key),
					"operator": types.StringValue(labelSelectorOperatorToString(expr.Operator)),
					"values":   types.ListValueMust(types.StringType, fromStringList(expr.Values)),
				},
			))
		}
		ls.MatchExpressions = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":      types.StringType,
					"operator": types.StringType,
					"values":   types.ListType{ElemType: types.StringType},
				},
			},
			exprs,
		)
	} else {
		ls.MatchExpressions = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":      types.StringType,
				"operator": types.StringType,
				"values":   types.ListType{ElemType: types.StringType},
			},
		})
	}

	return ls
}

// Helper function for disruption policy from proto.
func disruptionPolicyFromProto(disruption *apiv1.DisruptionPolicy) *DisruptionPolicy {
	dp := &DisruptionPolicy{
		ConsolidateAfter:              types.StringValue(disruption.ConsolidateAfter),
		ConsolidationPolicy:           types.StringValue(disruption.ConsolidationPolicy),
		ExpireAfter:                   types.StringValue(disruption.ExpireAfter),
		TtlSecondsAfterEmpty:          types.Int32Value(disruption.TtlSecondsAfterEmpty),
		TerminationGracePeriodSeconds: types.Int32Value(disruption.TerminationGracePeriodSeconds),
	}

	if len(disruption.Budgets) > 0 {
		budgets := make([]attr.Value, 0, len(disruption.Budgets))
		for _, budget := range disruption.Budgets {
			budgets = append(budgets, types.ObjectValueMust(
				map[string]attr.Type{
					"reasons":  types.ListType{ElemType: types.StringType},
					"nodes":    types.StringType,
					"schedule": types.StringType,
					"duration": types.StringType,
				},
				map[string]attr.Value{
					"reasons":  types.ListValueMust(types.StringType, fromStringList(budget.Reasons)),
					"nodes":    stringValue(budget.Nodes),
					"schedule": stringValue(budget.Schedule),
					"duration": stringValue(budget.Duration),
				},
			))
		}
		dp.Budgets = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"reasons":  types.ListType{ElemType: types.StringType},
					"nodes":    types.StringType,
					"schedule": types.StringType,
					"duration": types.StringType,
				},
			},
			budgets,
		)
	} else {
		dp.Budgets = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"reasons":  types.ListType{ElemType: types.StringType},
				"nodes":    types.StringType,
				"schedule": types.StringType,
				"duration": types.StringType,
			},
		})
	}

	return dp
}

// toProto for DisruptionPolicy.
func (dp *DisruptionPolicy) toProto(ctx context.Context, diags *diag.Diagnostics) *apiv1.DisruptionPolicy {
	disruption := &apiv1.DisruptionPolicy{
		ConsolidateAfter:              dp.ConsolidateAfter.ValueString(),
		ConsolidationPolicy:           dp.ConsolidationPolicy.ValueString(),
		ExpireAfter:                   dp.ExpireAfter.ValueString(),
		TtlSecondsAfterEmpty:          dp.TtlSecondsAfterEmpty.ValueInt32(),
		TerminationGracePeriodSeconds: dp.TerminationGracePeriodSeconds.ValueInt32(),
	}

	if !dp.Budgets.IsNull() && !dp.Budgets.IsUnknown() {
		// Manually extract budgets from types.Object (can't use getElementList due to nested types.List)
		var budgets []*apiv1.DisruptionBudget
		for _, elem := range dp.Budgets.Elements() {
			objVal, ok := elem.(types.Object)
			if !ok {
				continue
			}
			attrs := objVal.Attributes()

			var reasons []string
			if reasonsAttr, ok := attrs["reasons"].(types.List); ok && !reasonsAttr.IsNull() {
				var err error
				reasons, err = getStringList(ctx, reasonsAttr.Elements())
				if err != nil {
					diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert budget reasons: %s", err))
					return nil
				}
			}

			nodes := ""
			if nodesAttr, ok := attrs["nodes"].(types.String); ok && !nodesAttr.IsNull() {
				nodes = nodesAttr.ValueString()
			}

			schedule := ""
			if scheduleAttr, ok := attrs["schedule"].(types.String); ok && !scheduleAttr.IsNull() {
				schedule = scheduleAttr.ValueString()
			}

			duration := ""
			if durationAttr, ok := attrs["duration"].(types.String); ok && !durationAttr.IsNull() {
				duration = durationAttr.ValueString()
			}

			budgets = append(budgets, &apiv1.DisruptionBudget{
				Reasons:  reasons,
				Nodes:    nodes,
				Schedule: schedule,
				Duration: duration,
			})
		}
		disruption.Budgets = budgets
	}

	return disruption
}

// AWS Node Class conversion functions.
func (aws *AWSNodeClass) toProto(ctx context.Context, diags *diag.Diagnostics) *apiv1.AWSNodeClassSpec {
	spec := &apiv1.AWSNodeClassSpec{}

	// Subnet selector terms
	if !aws.SubnetSelectorTerms.IsNull() && !aws.SubnetSelectorTerms.IsUnknown() {
		// Manually iterate (can't use getElementList with types.Object due to pointer issues)
		var terms []*apiv1.SubnetSelectorTerm
		for _, elem := range aws.SubnetSelectorTerms.Elements() {
			objVal, ok := elem.(types.Object)
			if !ok {
				continue
			}
			attrs := objVal.Attributes()
			term := &apiv1.SubnetSelectorTerm{}
			if id, ok := attrs["id"].(types.String); ok && !id.IsNull() {
				term.Id = id.ValueString()
			}
			if tags, ok := attrs["tags"].(types.Map); ok && !tags.IsNull() {
				tagMap, err := getStringMap(ctx, tags.Elements())
				if err != nil {
					diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert subnet selector tags: %s", err))
					return nil
				}
				term.Tags = tagMap
			}
			terms = append(terms, term)
		}
		spec.SubnetSelectorTerms = terms
	}

	// Security group selector terms
	if !aws.SecurityGroupSelectorTerms.IsNull() && !aws.SecurityGroupSelectorTerms.IsUnknown() {
		// Manually iterate (can't use getElementList with types.Object due to pointer issues)
		var terms []*apiv1.SecurityGroupSelectorTerm
		for _, elem := range aws.SecurityGroupSelectorTerms.Elements() {
			objVal, ok := elem.(types.Object)
			if !ok {
				continue
			}
			attrs := objVal.Attributes()
			term := &apiv1.SecurityGroupSelectorTerm{}
			if id, ok := attrs["id"].(types.String); ok && !id.IsNull() {
				term.Id = id.ValueString()
			}
			if name, ok := attrs["name"].(types.String); ok && !name.IsNull() {
				term.Name = name.ValueString()
			}
			if tags, ok := attrs["tags"].(types.Map); ok && !tags.IsNull() {
				tagMap, err := getStringMap(ctx, tags.Elements())
				if err != nil {
					diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert security group selector tags: %s", err))
					return nil
				}
				term.Tags = tagMap
			}
			terms = append(terms, term)
		}
		spec.SecurityGroupSelectorTerms = terms
	}

	// AMI selector terms
	if !aws.AmiSelectorTerms.IsNull() && !aws.AmiSelectorTerms.IsUnknown() {
		// Manually iterate (can't use getElementList with types.Object due to pointer issues)
		var terms []*apiv1.AMISelectorTerm
		for _, elem := range aws.AmiSelectorTerms.Elements() {
			objVal, ok := elem.(types.Object)
			if !ok {
				continue
			}
			attrs := objVal.Attributes()
			term := &apiv1.AMISelectorTerm{}
			if id, ok := attrs["id"].(types.String); ok && !id.IsNull() {
				term.Id = id.ValueString()
			}
			if name, ok := attrs["name"].(types.String); ok && !name.IsNull() {
				term.Name = name.ValueString()
			}
			if owner, ok := attrs["owner"].(types.String); ok && !owner.IsNull() {
				term.Owner = owner.ValueString()
			}
			if alias, ok := attrs["alias"].(types.String); ok && !alias.IsNull() {
				term.Alias = alias.ValueString()
			}
			if tags, ok := attrs["tags"].(types.Map); ok && !tags.IsNull() {
				tagMap, err := getStringMap(ctx, tags.Elements())
				if err != nil {
					diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert AMI selector tags: %s", err))
					return nil
				}
				term.Tags = tagMap
			}
			terms = append(terms, term)
		}
		spec.AmiSelectorTerms = terms
	}

	// Simple string fields
	if !aws.AmiFamily.IsNull() {
		val := aws.AmiFamily.ValueString()
		spec.AmiFamily = &val
	}
	if !aws.UserData.IsNull() {
		val := aws.UserData.ValueString()
		spec.UserData = &val
	}
	if !aws.Role.IsNull() {
		val := aws.Role.ValueString()
		spec.Role = &val
	}
	if !aws.InstanceProfile.IsNull() {
		val := aws.InstanceProfile.ValueString()
		spec.InstanceProfile = &val
	}

	// Tags
	if !aws.Tags.IsNull() {
		tags, err := getStringMap(ctx, aws.Tags.Elements())
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert AWS tags: %s", err))
			return nil
		}
		spec.Tags = tags
	}

	// Block device mappings
	if !aws.BlockDeviceMappings.IsNull() && !aws.BlockDeviceMappings.IsUnknown() {
		// Manually iterate (can't use getElementList with types.Object due to pointer issues)
		var mappings []*apiv1.BlockDeviceMapping
		for _, elem := range aws.BlockDeviceMappings.Elements() {
			objVal, ok := elem.(types.Object)
			if !ok {
				continue
			}
			attrs := objVal.Attributes()
			mapping := &apiv1.BlockDeviceMapping{}

			// Device name (pointer)
			if deviceName, ok := attrs["device_name"].(types.String); ok && !deviceName.IsNull() {
				val := deviceName.ValueString()
				mapping.DeviceName = &val
			}

			// EBS configuration (nested)
			if ebsObj, ok := attrs["ebs"].(types.Object); ok && !ebsObj.IsNull() {
				ebsAttrs := ebsObj.Attributes()
				ebs := &apiv1.BlockDevice{}

				if volumeSize, ok := ebsAttrs["volume_size"].(types.String); ok && !volumeSize.IsNull() {
					val := volumeSize.ValueString()
					ebs.VolumeSize = &val
				}
				if volumeType, ok := ebsAttrs["volume_type"].(types.String); ok && !volumeType.IsNull() {
					val := volumeType.ValueString()
					ebs.VolumeType = &val
				}
				if iops, ok := ebsAttrs["iops"].(types.Int64); ok && !iops.IsNull() {
					val := iops.ValueInt64()
					ebs.Iops = &val
				}
				if throughput, ok := ebsAttrs["throughput"].(types.Int64); ok && !throughput.IsNull() {
					val := throughput.ValueInt64()
					ebs.Throughput = &val
				}
				if kmsKeyId, ok := ebsAttrs["kms_key_id"].(types.String); ok && !kmsKeyId.IsNull() {
					val := kmsKeyId.ValueString()
					ebs.KmsKeyId = &val
				}
				if snapshotId, ok := ebsAttrs["snapshot_id"].(types.String); ok && !snapshotId.IsNull() {
					val := snapshotId.ValueString()
					ebs.SnapshotId = &val
				}
				if deleteOnTermination, ok := ebsAttrs["delete_on_termination"].(types.Bool); ok && !deleteOnTermination.IsNull() {
					val := deleteOnTermination.ValueBool()
					ebs.DeleteOnTermination = &val
				}
				if encrypted, ok := ebsAttrs["encrypted"].(types.Bool); ok && !encrypted.IsNull() {
					val := encrypted.ValueBool()
					ebs.Encrypted = &val
				}

				mapping.Ebs = ebs
			}

			mappings = append(mappings, mapping)
		}
		spec.BlockDeviceMappings = mappings
	}

	// Instance store policy
	if !aws.InstanceStorePolicy.IsNull() {
		policy := instanceStorePolicyFromString(aws.InstanceStorePolicy.ValueString())
		spec.InstanceStorePolicy = &policy
	}

	// Boolean fields
	if !aws.DetailedMonitoring.IsNull() {
		val := aws.DetailedMonitoring.ValueBool()
		spec.DetailedMonitoring = &val
	}
	if !aws.AssociatePublicIpAddress.IsNull() {
		val := aws.AssociatePublicIpAddress.ValueBool()
		spec.AssociatePublicIpAddress = &val
	}

	// Metadata options
	if aws.MetadataOptions != nil {
		metadataOpts := &apiv1.MetadataOptions{}
		if !aws.MetadataOptions.HttpEndpoint.IsNull() {
			val := aws.MetadataOptions.HttpEndpoint.ValueString()
			metadataOpts.HttpEndpoint = &val
		}
		if !aws.MetadataOptions.HttpProtocolIpv6.IsNull() {
			val := aws.MetadataOptions.HttpProtocolIpv6.ValueString()
			metadataOpts.HttpProtocolIpv6 = &val
		}
		if !aws.MetadataOptions.HttpPutResponseHopLimit.IsNull() {
			val := aws.MetadataOptions.HttpPutResponseHopLimit.ValueInt64()
			metadataOpts.HttpPutResponseHopLimit = &val
		}
		if !aws.MetadataOptions.HttpTokens.IsNull() {
			val := aws.MetadataOptions.HttpTokens.ValueString()
			metadataOpts.HttpTokens = &val
		}
		spec.MetadataOptions = metadataOpts
	}

	return spec
}

func awsNodeClassFromProto(spec *apiv1.AWSNodeClassSpec) *AWSNodeClass {
	aws := &AWSNodeClass{}

	// Subnet selector terms
	if len(spec.SubnetSelectorTerms) > 0 {
		terms := make([]attr.Value, 0, len(spec.SubnetSelectorTerms))
		for _, term := range spec.SubnetSelectorTerms {
			termAttrs := map[string]attr.Value{
				"id": stringValue(term.Id),
			}
			if term.Tags != nil {
				termAttrs["tags"] = types.MapValueMust(types.StringType, fromStringMap(term.Tags))
			} else {
				termAttrs["tags"] = types.MapNull(types.StringType)
			}
			terms = append(terms, types.ObjectValueMust(
				map[string]attr.Type{
					"id":   types.StringType,
					"tags": types.MapType{ElemType: types.StringType},
				},
				termAttrs,
			))
		}
		aws.SubnetSelectorTerms = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":   types.StringType,
					"tags": types.MapType{ElemType: types.StringType},
				},
			},
			terms,
		)
	} else {
		aws.SubnetSelectorTerms = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":   types.StringType,
				"tags": types.MapType{ElemType: types.StringType},
			},
		})
	}

	// Security group selector terms
	if len(spec.SecurityGroupSelectorTerms) > 0 {
		terms := make([]attr.Value, 0, len(spec.SecurityGroupSelectorTerms))
		for _, term := range spec.SecurityGroupSelectorTerms {
			termAttrs := map[string]attr.Value{
				"id":   stringValue(term.Id),
				"name": stringValue(term.Name),
			}
			if term.Tags != nil {
				termAttrs["tags"] = types.MapValueMust(types.StringType, fromStringMap(term.Tags))
			} else {
				termAttrs["tags"] = types.MapNull(types.StringType)
			}
			terms = append(terms, types.ObjectValueMust(
				map[string]attr.Type{
					"id":   types.StringType,
					"name": types.StringType,
					"tags": types.MapType{ElemType: types.StringType},
				},
				termAttrs,
			))
		}
		aws.SecurityGroupSelectorTerms = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":   types.StringType,
					"name": types.StringType,
					"tags": types.MapType{ElemType: types.StringType},
				},
			},
			terms,
		)
	} else {
		aws.SecurityGroupSelectorTerms = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":   types.StringType,
				"name": types.StringType,
				"tags": types.MapType{ElemType: types.StringType},
			},
		})
	}

	// AMI selector terms
	if len(spec.AmiSelectorTerms) > 0 {
		terms := make([]attr.Value, 0, len(spec.AmiSelectorTerms))
		for _, term := range spec.AmiSelectorTerms {
			termAttrs := map[string]attr.Value{
				"id":    stringValue(term.Id),
				"name":  stringValue(term.Name),
				"owner": stringValue(term.Owner),
				"alias": stringValue(term.Alias),
			}
			if term.Tags != nil {
				termAttrs["tags"] = types.MapValueMust(types.StringType, fromStringMap(term.Tags))
			} else {
				termAttrs["tags"] = types.MapNull(types.StringType)
			}
			terms = append(terms, types.ObjectValueMust(
				map[string]attr.Type{
					"id":    types.StringType,
					"name":  types.StringType,
					"owner": types.StringType,
					"alias": types.StringType,
					"tags":  types.MapType{ElemType: types.StringType},
				},
				termAttrs,
			))
		}
		aws.AmiSelectorTerms = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":    types.StringType,
					"name":  types.StringType,
					"owner": types.StringType,
					"alias": types.StringType,
					"tags":  types.MapType{ElemType: types.StringType},
				},
			},
			terms,
		)
	} else {
		aws.AmiSelectorTerms = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":    types.StringType,
				"name":  types.StringType,
				"owner": types.StringType,
				"alias": types.StringType,
				"tags":  types.MapType{ElemType: types.StringType},
			},
		})
	}

	// Simple fields
	aws.AmiFamily = stringPointerValue(spec.AmiFamily)
	aws.UserData = stringPointerValue(spec.UserData)
	aws.Role = stringPointerValue(spec.Role)
	aws.InstanceProfile = stringPointerValue(spec.InstanceProfile)

	// Tags
	if spec.Tags != nil {
		aws.Tags = types.MapValueMust(types.StringType, fromStringMap(spec.Tags))
	} else {
		aws.Tags = types.MapNull(types.StringType)
	}

	// Block device mappings
	if len(spec.BlockDeviceMappings) > 0 {
		mappings := make([]attr.Value, 0, len(spec.BlockDeviceMappings))
		for _, mapping := range spec.BlockDeviceMappings {
			mappingAttrs := map[string]attr.Value{
				"device_name": stringPointerValue(mapping.DeviceName),
			}

			// EBS configuration (nested)
			if mapping.Ebs != nil {
				ebsAttrs := map[string]attr.Value{
					"volume_size": stringPointerValue(mapping.Ebs.VolumeSize),
					"volume_type": stringPointerValue(mapping.Ebs.VolumeType),
					"kms_key_id":  stringPointerValue(mapping.Ebs.KmsKeyId),
					"snapshot_id": stringPointerValue(mapping.Ebs.SnapshotId),
				}

				if mapping.Ebs.Iops != nil {
					ebsAttrs["iops"] = types.Int64Value(*mapping.Ebs.Iops)
				} else {
					ebsAttrs["iops"] = types.Int64Null()
				}
				if mapping.Ebs.Throughput != nil {
					ebsAttrs["throughput"] = types.Int64Value(*mapping.Ebs.Throughput)
				} else {
					ebsAttrs["throughput"] = types.Int64Null()
				}
				if mapping.Ebs.DeleteOnTermination != nil {
					ebsAttrs["delete_on_termination"] = types.BoolValue(*mapping.Ebs.DeleteOnTermination)
				} else {
					ebsAttrs["delete_on_termination"] = types.BoolNull()
				}
				if mapping.Ebs.Encrypted != nil {
					ebsAttrs["encrypted"] = types.BoolValue(*mapping.Ebs.Encrypted)
				} else {
					ebsAttrs["encrypted"] = types.BoolNull()
				}

				mappingAttrs["ebs"] = types.ObjectValueMust(
					map[string]attr.Type{
						"volume_size":           types.StringType,
						"volume_type":           types.StringType,
						"iops":                  types.Int64Type,
						"throughput":            types.Int64Type,
						"kms_key_id":            types.StringType,
						"snapshot_id":           types.StringType,
						"delete_on_termination": types.BoolType,
						"encrypted":             types.BoolType,
					},
					ebsAttrs,
				)
			} else {
				mappingAttrs["ebs"] = types.ObjectNull(map[string]attr.Type{
					"volume_size":           types.StringType,
					"volume_type":           types.StringType,
					"iops":                  types.Int64Type,
					"throughput":            types.Int64Type,
					"kms_key_id":            types.StringType,
					"snapshot_id":           types.StringType,
					"delete_on_termination": types.BoolType,
					"encrypted":             types.BoolType,
				})
			}

			mappings = append(mappings, types.ObjectValueMust(
				map[string]attr.Type{
					"device_name": types.StringType,
					"ebs": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"volume_size":           types.StringType,
							"volume_type":           types.StringType,
							"iops":                  types.Int64Type,
							"throughput":            types.Int64Type,
							"kms_key_id":            types.StringType,
							"snapshot_id":           types.StringType,
							"delete_on_termination": types.BoolType,
							"encrypted":             types.BoolType,
						},
					},
				},
				mappingAttrs,
			))
		}
		aws.BlockDeviceMappings = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"device_name": types.StringType,
					"ebs": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"volume_size":           types.StringType,
							"volume_type":           types.StringType,
							"iops":                  types.Int64Type,
							"throughput":            types.Int64Type,
							"kms_key_id":            types.StringType,
							"snapshot_id":           types.StringType,
							"delete_on_termination": types.BoolType,
							"encrypted":             types.BoolType,
						},
					},
				},
			},
			mappings,
		)
	} else {
		aws.BlockDeviceMappings = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"device_name": types.StringType,
				"ebs": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"volume_size":           types.StringType,
						"volume_type":           types.StringType,
						"iops":                  types.Int64Type,
						"throughput":            types.Int64Type,
						"kms_key_id":            types.StringType,
						"snapshot_id":           types.StringType,
						"delete_on_termination": types.BoolType,
						"encrypted":             types.BoolType,
					},
				},
			},
		})
	}

	// Instance store policy
	if spec.InstanceStorePolicy != nil {
		aws.InstanceStorePolicy = types.StringValue(instanceStorePolicyToString(*spec.InstanceStorePolicy))
	} else {
		aws.InstanceStorePolicy = types.StringNull()
	}

	// Boolean fields
	if spec.DetailedMonitoring != nil {
		aws.DetailedMonitoring = types.BoolValue(*spec.DetailedMonitoring)
	} else {
		aws.DetailedMonitoring = types.BoolNull()
	}
	if spec.AssociatePublicIpAddress != nil {
		aws.AssociatePublicIpAddress = types.BoolValue(*spec.AssociatePublicIpAddress)
	} else {
		aws.AssociatePublicIpAddress = types.BoolNull()
	}

	// Metadata options
	if spec.MetadataOptions != nil {
		metadataOpts := &MetadataOptions{}
		if spec.MetadataOptions.HttpEndpoint != nil {
			metadataOpts.HttpEndpoint = types.StringValue(*spec.MetadataOptions.HttpEndpoint)
		} else {
			metadataOpts.HttpEndpoint = types.StringNull()
		}
		if spec.MetadataOptions.HttpProtocolIpv6 != nil {
			metadataOpts.HttpProtocolIpv6 = types.StringValue(*spec.MetadataOptions.HttpProtocolIpv6)
		} else {
			metadataOpts.HttpProtocolIpv6 = types.StringNull()
		}
		if spec.MetadataOptions.HttpPutResponseHopLimit != nil {
			metadataOpts.HttpPutResponseHopLimit = types.Int64Value(*spec.MetadataOptions.HttpPutResponseHopLimit)
		} else {
			metadataOpts.HttpPutResponseHopLimit = types.Int64Null()
		}
		if spec.MetadataOptions.HttpTokens != nil {
			metadataOpts.HttpTokens = types.StringValue(*spec.MetadataOptions.HttpTokens)
		} else {
			metadataOpts.HttpTokens = types.StringNull()
		}
		aws.MetadataOptions = metadataOpts
	}

	return aws
}

// Azure Node Class conversion functions.
func (azure *AzureNodeClass) toProto(ctx context.Context, diags *diag.Diagnostics) *apiv1.AzureNodeClassSpec {
	spec := &apiv1.AzureNodeClassSpec{}

	if !azure.VnetSubnetId.IsNull() {
		val := azure.VnetSubnetId.ValueString()
		spec.VnetSubnetId = &val
	}
	if !azure.OsDiskSizeGb.IsNull() {
		val := azure.OsDiskSizeGb.ValueInt32()
		spec.OsDiskSizeGb = &val
	}
	if !azure.ImageFamily.IsNull() {
		val := azure.ImageFamily.ValueString()
		spec.ImageFamily = &val
	}
	if !azure.FipsMode.IsNull() {
		val := azure.FipsMode.ValueString()
		spec.FipsMode = &val
	}

	if !azure.Tags.IsNull() {
		tags, err := getStringMap(ctx, azure.Tags.Elements())
		if err != nil {
			diags.AddError("Conversion Error", fmt.Sprintf("Unable to convert Azure tags: %s", err))
			return nil
		}
		spec.Tags = tags
	}

	if !azure.MaxPods.IsNull() {
		val := azure.MaxPods.ValueInt32()
		spec.MaxPods = &val
	}

	return spec
}

// Helper to check if AWS spec is empty (all fields are nil/empty).
func isAWSSpecEmpty(spec *apiv1.AWSNodeClassSpec) bool {
	if spec == nil {
		return true
	}
	return len(spec.SubnetSelectorTerms) == 0 &&
		len(spec.SecurityGroupSelectorTerms) == 0 &&
		len(spec.AmiSelectorTerms) == 0 &&
		spec.AmiFamily == nil &&
		spec.UserData == nil &&
		spec.Role == nil &&
		spec.InstanceProfile == nil &&
		spec.Tags == nil &&
		len(spec.BlockDeviceMappings) == 0 &&
		spec.InstanceStorePolicy == nil &&
		spec.DetailedMonitoring == nil &&
		spec.AssociatePublicIpAddress == nil &&
		spec.MetadataOptions == nil
}

// Helper to check if Azure spec is empty (all fields are nil).
func isAzureSpecEmpty(spec *apiv1.AzureNodeClassSpec) bool {
	if spec == nil {
		return true
	}
	return spec.VnetSubnetId == nil &&
		spec.OsDiskSizeGb == nil &&
		spec.ImageFamily == nil &&
		spec.FipsMode == nil &&
		spec.Tags == nil &&
		spec.MaxPods == nil
}

func azureNodeClassFromProto(spec *apiv1.AzureNodeClassSpec) *AzureNodeClass {
	azure := &AzureNodeClass{}

	azure.VnetSubnetId = stringPointerValue(spec.VnetSubnetId)

	if spec.OsDiskSizeGb != nil {
		azure.OsDiskSizeGb = types.Int32Value(*spec.OsDiskSizeGb)
	} else {
		azure.OsDiskSizeGb = types.Int32Null()
	}

	azure.ImageFamily = stringPointerValue(spec.ImageFamily)
	azure.FipsMode = stringPointerValue(spec.FipsMode)

	if spec.Tags != nil {
		azure.Tags = types.MapValueMust(types.StringType, fromStringMap(spec.Tags))
	} else {
		azure.Tags = types.MapNull(types.StringType)
	}

	if spec.MaxPods != nil {
		azure.MaxPods = types.Int32Value(*spec.MaxPods)
	} else {
		azure.MaxPods = types.Int32Null()
	}

	return azure
}

// Helper functions for enum conversions.
//
//nolint:unparam // Only RAID0 is currently supported, but function provides extensibility
func instanceStorePolicyFromString(s string) apiv1.InstanceStorePolicy {
	switch s {
	case "RAID0":
		return apiv1.InstanceStorePolicy_INSTANCE_STORE_POLICY_RAID0
	default:
		return apiv1.InstanceStorePolicy_INSTANCE_STORE_POLICY_RAID0 // Only RAID0 is supported
	}
}

func instanceStorePolicyToString(policy apiv1.InstanceStorePolicy) string {
	switch policy {
	case apiv1.InstanceStorePolicy_INSTANCE_STORE_POLICY_RAID0:
		return "RAID0"
	default:
		return ""
	}
}

func labelSelectorOperatorToString(op apiv1.LabelSelectorOperator) string {
	switch op {
	case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_IN:
		return "In"
	case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_NOT_IN:
		return "NotIn"
	case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_EXISTS:
		return "Exists"
	case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_DOES_NOT_EXIST:
		return "DoesNotExist"
	case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_GT:
		return "Gt"
	case apiv1.LabelSelectorOperator_LABEL_SELECTOR_OPERATOR_LT:
		return "Lt"
	default:
		return ""
	}
}
