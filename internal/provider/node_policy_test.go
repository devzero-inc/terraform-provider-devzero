package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNodePolicyResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Instantiate the resource and call Schema
	nodePolicyResource := NewNodePolicyResource()
	nodePolicyResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	// Validate the schema
	validateNodePolicySchema(t, resp.Schema)
}

func TestNodePolicyResourceModel(t *testing.T) {
	t.Parallel()

	// Test Taint
	t.Run("Taint", func(t *testing.T) {
		taint := Taint{
			Key:    types.StringValue("workload-type"),
			Value:  types.StringValue("batch"),
			Effect: types.StringValue("NoSchedule"),
		}

		if taint.Key.ValueString() != "workload-type" {
			t.Errorf("Expected key to be 'workload-type', got %s", taint.Key.ValueString())
		}
		if taint.Value.ValueString() != "batch" {
			t.Errorf("Expected value to be 'batch', got %s", taint.Value.ValueString())
		}
		if taint.Effect.ValueString() != "NoSchedule" {
			t.Errorf("Expected effect to be 'NoSchedule', got %s", taint.Effect.ValueString())
		}
	})

	// Test ResourceLimits
	t.Run("ResourceLimits", func(t *testing.T) {
		limits := &ResourceLimits{
			Cpu:    types.StringValue("100"),
			Memory: types.StringValue("512Gi"),
		}

		if limits.Cpu.ValueString() != "100" {
			t.Errorf("Expected CPU limit to be '100', got %s", limits.Cpu.ValueString())
		}
		if limits.Memory.ValueString() != "512Gi" {
			t.Errorf("Expected Memory limit to be '512Gi', got %s", limits.Memory.ValueString())
		}
	})

	// Test DisruptionBudget
	t.Run("DisruptionBudget", func(t *testing.T) {
		budget := DisruptionBudget{
			Reasons:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Underutilized"), types.StringValue("Empty")}),
			Nodes:    types.StringValue("10%"),
			Schedule: types.StringValue("0 2 * * *"),
			Duration: types.StringValue("1h30m"),
		}

		if budget.Nodes.ValueString() != "10%" {
			t.Errorf("Expected nodes to be '10%%', got %s", budget.Nodes.ValueString())
		}
		if budget.Schedule.ValueString() != "0 2 * * *" {
			t.Errorf("Expected schedule to be '0 2 * * *', got %s", budget.Schedule.ValueString())
		}
		if budget.Duration.ValueString() != "1h30m" {
			t.Errorf("Expected duration to be '1h30m', got %s", budget.Duration.ValueString())
		}
	})

	// Test DisruptionPolicy
	t.Run("DisruptionPolicy", func(t *testing.T) {
		policy := &DisruptionPolicy{
			ConsolidateAfter:              types.StringValue("5m"),
			ConsolidationPolicy:           types.StringValue("WhenEmptyOrUnderutilized"),
			ExpireAfter:                   types.StringValue("720h"),
			TtlSecondsAfterEmpty:          types.Int32Value(300),
			TerminationGracePeriodSeconds: types.Int32Value(30),
			Budgets:                       types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}

		if policy.ConsolidateAfter.ValueString() != "5m" {
			t.Errorf("Expected consolidate_after to be '5m', got %s", policy.ConsolidateAfter.ValueString())
		}
		if policy.ConsolidationPolicy.ValueString() != "WhenEmptyOrUnderutilized" {
			t.Errorf("Expected consolidation_policy to be 'WhenEmptyOrUnderutilized', got %s", policy.ConsolidationPolicy.ValueString())
		}
		if policy.ExpireAfter.ValueString() != "720h" {
			t.Errorf("Expected expire_after to be '720h', got %s", policy.ExpireAfter.ValueString())
		}
		if policy.TtlSecondsAfterEmpty.ValueInt32() != 300 {
			t.Errorf("Expected ttl_seconds_after_empty to be 300, got %d", policy.TtlSecondsAfterEmpty.ValueInt32())
		}
	})

	// Test AWSNodeClass
	t.Run("AWSNodeClass", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			AmiFamily:                  types.StringValue("AL2"),
			UserData:                   types.StringValue("#!/bin/bash\necho 'test'"),
			Role:                       types.StringValue("KarpenterNodeRole"),
			InstanceProfile:            types.StringValue("KarpenterNodeInstanceProfile"),
			Tags:                       types.MapValueMust(types.StringType, map[string]attr.Value{"Environment": types.StringValue("production")}),
			InstanceStorePolicy:        types.StringValue("RAID0"),
			DetailedMonitoring:         types.BoolValue(true),
			AssociatePublicIpAddress:   types.BoolValue(false),
			SubnetSelectorTerms:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:           types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}

		if awsConfig.AmiFamily.ValueString() != "AL2" {
			t.Errorf("Expected ami_family to be 'AL2', got %s", awsConfig.AmiFamily.ValueString())
		}
		if awsConfig.Role.ValueString() != "KarpenterNodeRole" {
			t.Errorf("Expected role to be 'KarpenterNodeRole', got %s", awsConfig.Role.ValueString())
		}
		if awsConfig.InstanceStorePolicy.ValueString() != "RAID0" {
			t.Errorf("Expected instance_store_policy to be 'RAID0', got %s", awsConfig.InstanceStorePolicy.ValueString())
		}
		if !awsConfig.DetailedMonitoring.ValueBool() {
			t.Error("Expected detailed_monitoring to be true")
		}
	})

	// Test AzureNodeClass
	t.Run("AzureNodeClass", func(t *testing.T) {
		azureConfig := &AzureNodeClass{
			VnetSubnetId: types.StringValue("/subscriptions/xxx/resourceGroups/yyy/providers/Microsoft.Network/virtualNetworks/zzz/subnets/aaa"),
			OsDiskSizeGb: types.Int32Value(128),
			ImageFamily:  types.StringValue("Ubuntu2204"),
			FipsMode:     types.StringValue("Disabled"),
			Tags:         types.MapValueMust(types.StringType, map[string]attr.Value{"Environment": types.StringValue("production")}),
			MaxPods:      types.Int32Value(110),
		}

		if azureConfig.VnetSubnetId.ValueString() == "" {
			t.Error("Expected VnetSubnetId to be non-empty")
		}
		if azureConfig.OsDiskSizeGb.ValueInt32() != 128 {
			t.Errorf("Expected os_disk_size_gb to be 128, got %d", azureConfig.OsDiskSizeGb.ValueInt32())
		}
		if azureConfig.ImageFamily.ValueString() != "Ubuntu2204" {
			t.Errorf("Expected image_family to be 'Ubuntu2204', got %s", azureConfig.ImageFamily.ValueString())
		}
		if azureConfig.MaxPods.ValueInt32() != 110 {
			t.Errorf("Expected max_pods to be 110, got %d", azureConfig.MaxPods.ValueInt32())
		}
	})

	// Test RawKarpenterSpec
	t.Run("RawKarpenterSpec", func(t *testing.T) {
		rawSpec := RawKarpenterSpec{
			NodepoolYaml:  types.StringValue("apiVersion: karpenter.sh/v1\nkind: NodePool"),
			NodeclassYaml: types.StringValue("apiVersion: karpenter.k8s.aws/v1\nkind: EC2NodeClass"),
		}

		if rawSpec.NodepoolYaml.ValueString() == "" {
			t.Error("Expected NodepoolYaml to be non-empty")
		}
		if rawSpec.NodeclassYaml.ValueString() == "" {
			t.Error("Expected NodeclassYaml to be non-empty")
		}
	})

	// Test InstanceStorePolicy enum conversions
	t.Run("InstanceStorePolicyConversions", func(t *testing.T) {
		// Test fromString
		policy := instanceStorePolicyFromString("RAID0")
		if policy != 0 { // RAID0 = 0
			t.Errorf("Expected RAID0 policy (0), got %d", policy)
		}

		// Test toString
		result := instanceStorePolicyToString(0) // RAID0
		if result != "RAID0" {
			t.Errorf("Expected 'RAID0', got %s", result)
		}
	})

	// Test LabelSelectorOperator conversions
	t.Run("LabelSelectorOperatorConversions", func(t *testing.T) {
		// Test toString
		result := labelSelectorOperatorToString(1) // IN
		if result != "In" {
			t.Errorf("Expected 'In', got %s", result)
		}

		result = labelSelectorOperatorToString(2) // NOT_IN
		if result != "NotIn" {
			t.Errorf("Expected 'NotIn', got %s", result)
		}

		result = labelSelectorOperatorToString(3) // EXISTS
		if result != "Exists" {
			t.Errorf("Expected 'Exists', got %s", result)
		}

		result = labelSelectorOperatorToString(4) // DOES_NOT_EXIST
		if result != "DoesNotExist" {
			t.Errorf("Expected 'DoesNotExist', got %s", result)
		}

		result = labelSelectorOperatorToString(5) // GT
		if result != "Gt" {
			t.Errorf("Expected 'Gt', got %s", result)
		}

		result = labelSelectorOperatorToString(6) // LT
		if result != "Lt" {
			t.Errorf("Expected 'Lt', got %s", result)
		}
	})

	// Test LabelSelector with MatchExpressions (as they come from Terraform)
	t.Run("LabelSelector_WithMatchExpressions", func(t *testing.T) {
		selector := &LabelSelector{
			MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"app": types.StringValue("api"),
			}),
			MatchExpressions: types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"key":      types.StringType,
						"operator": types.StringType,
						"values":   types.ListType{ElemType: types.StringType},
					},
				},
				[]attr.Value{
					types.ObjectValueMust(
						map[string]attr.Type{
							"key":      types.StringType,
							"operator": types.StringType,
							"values":   types.ListType{ElemType: types.StringType},
						},
						map[string]attr.Value{
							"key":      types.StringValue("instanceGenerations"),
							"operator": types.StringValue("Gt"),
							"values":   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("4")}),
						},
					),
				},
			),
		}

		// Test toProto
		ctx := context.Background()
		proto, err := selector.toProto(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.MatchLabels) != 1 {
			t.Errorf("Expected 1 match label, got %d", len(proto.MatchLabels))
		}
		if proto.MatchLabels["app"] != "api" {
			t.Errorf("Expected app=api, got %s", proto.MatchLabels["app"])
		}
		if len(proto.MatchExpressions) != 1 {
			t.Fatalf("Expected 1 match expression, got %d", len(proto.MatchExpressions))
		}
		expr := proto.MatchExpressions[0]
		if expr.Key != "instanceGenerations" {
			t.Errorf("Expected key=instanceGenerations, got %s", expr.Key)
		}
		if expr.Operator != 5 { // GT = 5
			t.Errorf("Expected operator=GT (5), got %d", expr.Operator)
		}
		if len(expr.Values) != 1 {
			t.Errorf("Expected 1 value, got %d", len(expr.Values))
		}
		if expr.Values[0] != "4" {
			t.Errorf("Expected value '4', got %s", expr.Values[0])
		}
	})

	// Test DisruptionPolicy with nested budgets (full toProto conversion)
	t.Run("DisruptionPolicy_WithBudgets", func(t *testing.T) {
		policy := &DisruptionPolicy{
			ConsolidateAfter:              types.StringValue("5m"),
			ConsolidationPolicy:           types.StringValue("WhenEmptyOrUnderutilized"),
			ExpireAfter:                   types.StringValue("720h"),
			TtlSecondsAfterEmpty:          types.Int32Value(300),
			TerminationGracePeriodSeconds: types.Int32Value(30),
			Budgets: types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"reasons":  types.ListType{ElemType: types.StringType},
						"nodes":    types.StringType,
						"schedule": types.StringType,
						"duration": types.StringType,
					},
				},
				[]attr.Value{
					types.ObjectValueMust(
						map[string]attr.Type{
							"reasons":  types.ListType{ElemType: types.StringType},
							"nodes":    types.StringType,
							"schedule": types.StringType,
							"duration": types.StringType,
						},
						map[string]attr.Value{
							"reasons":  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Underutilized"), types.StringValue("Empty")}),
							"nodes":    types.StringValue("10%"),
							"schedule": types.StringValue("0 2 * * *"),
							"duration": types.StringValue("1h30m"),
						},
					),
				},
			),
		}

		// Test toProto
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := policy.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.Budgets) != 1 {
			t.Fatalf("Expected 1 budget, got %d", len(proto.Budgets))
		}
		budget := proto.Budgets[0]
		if len(budget.Reasons) != 2 {
			t.Errorf("Expected 2 reasons, got %d", len(budget.Reasons))
		}
		if budget.Reasons[0] != "Underutilized" {
			t.Errorf("Expected reason 'Underutilized', got %s", budget.Reasons[0])
		}
		if budget.Nodes != "10%" {
			t.Errorf("Expected nodes '10%%', got %s", budget.Nodes)
		}
	})

	// Test SubnetSelectorTerms conversion
	t.Run("SubnetSelectorTerms_ToProto", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			SubnetSelectorTerms: types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"tags": types.MapType{ElemType: types.StringType},
						"id":   types.StringType,
					},
				},
				[]attr.Value{
					types.ObjectValueMust(
						map[string]attr.Type{
							"tags": types.MapType{ElemType: types.StringType},
							"id":   types.StringType,
						},
						map[string]attr.Value{
							"tags": types.MapValueMust(types.StringType, map[string]attr.Value{
								"karpenter.sh/discovery": types.StringValue("my-cluster"),
							}),
							"id": types.StringValue("subnet-12345"),
						},
					),
				},
			),
			AmiFamily:                  types.StringValue("AL2"),
			SecurityGroupSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:           types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}

		// Test toProto
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.SubnetSelectorTerms) != 1 {
			t.Fatalf("Expected 1 subnet selector term, got %d", len(proto.SubnetSelectorTerms))
		}
		term := proto.SubnetSelectorTerms[0]
		if term.Tags["karpenter.sh/discovery"] != "my-cluster" {
			t.Errorf("Expected tag 'my-cluster', got %s", term.Tags["karpenter.sh/discovery"])
		}
		if term.Id != "subnet-12345" {
			t.Errorf("Expected id 'subnet-12345', got %s", term.Id)
		}
	})

	// Test SecurityGroupSelectorTerms conversion
	t.Run("SecurityGroupSelectorTerms_ToProto", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			SecurityGroupSelectorTerms: types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"tags": types.MapType{ElemType: types.StringType},
						"id":   types.StringType,
						"name": types.StringType,
					},
				},
				[]attr.Value{
					types.ObjectValueMust(
						map[string]attr.Type{
							"tags": types.MapType{ElemType: types.StringType},
							"id":   types.StringType,
							"name": types.StringType,
						},
						map[string]attr.Value{
							"tags": types.MapValueMust(types.StringType, map[string]attr.Value{
								"karpenter.sh/discovery": types.StringValue("my-cluster"),
							}),
							"id":   types.StringValue("sg-12345"),
							"name": types.StringValue("my-security-group"),
						},
					),
				},
			),
			AmiFamily:           types.StringValue("AL2"),
			SubnetSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:    types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}

		// Test toProto
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.SecurityGroupSelectorTerms) != 1 {
			t.Fatalf("Expected 1 security group selector term, got %d", len(proto.SecurityGroupSelectorTerms))
		}
		term := proto.SecurityGroupSelectorTerms[0]
		if term.Tags["karpenter.sh/discovery"] != "my-cluster" {
			t.Errorf("Expected tag 'my-cluster', got %s", term.Tags["karpenter.sh/discovery"])
		}
		if term.Id != "sg-12345" {
			t.Errorf("Expected id 'sg-12345', got %s", term.Id)
		}
		if term.Name != "my-security-group" {
			t.Errorf("Expected name 'my-security-group', got %s", term.Name)
		}
	})

	// Test AmiSelectorTerms conversion
	t.Run("AmiSelectorTerms_ToProto", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			AmiSelectorTerms: types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"tags":  types.MapType{ElemType: types.StringType},
						"id":    types.StringType,
						"name":  types.StringType,
						"owner": types.StringType,
						"alias": types.StringType,
					},
				},
				[]attr.Value{
					types.ObjectValueMust(
						map[string]attr.Type{
							"tags":  types.MapType{ElemType: types.StringType},
							"id":    types.StringType,
							"name":  types.StringType,
							"owner": types.StringType,
							"alias": types.StringType,
						},
						map[string]attr.Value{
							"tags": types.MapValueMust(types.StringType, map[string]attr.Value{
								"Environment": types.StringValue("production"),
							}),
							"id":    types.StringValue("ami-12345"),
							"name":  types.StringValue("my-ami"),
							"owner": types.StringValue("123456789012"),
							"alias": types.StringValue("al2/stable"),
						},
					),
				},
			),
			AmiFamily:                  types.StringValue("AL2"),
			SubnetSelectorTerms:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}

		// Test toProto
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.AmiSelectorTerms) != 1 {
			t.Fatalf("Expected 1 AMI selector term, got %d", len(proto.AmiSelectorTerms))
		}
		term := proto.AmiSelectorTerms[0]
		if term.Tags["Environment"] != "production" {
			t.Errorf("Expected tag 'production', got %s", term.Tags["Environment"])
		}
		if term.Id != "ami-12345" {
			t.Errorf("Expected id 'ami-12345', got %s", term.Id)
		}
		if term.Name != "my-ami" {
			t.Errorf("Expected name 'my-ami', got %s", term.Name)
		}
		if term.Owner != "123456789012" {
			t.Errorf("Expected owner '123456789012', got %s", term.Owner)
		}
		if term.Alias != "al2/stable" {
			t.Errorf("Expected alias 'al2/stable', got %s", term.Alias)
		}
	})

	// Test BlockDeviceMappings conversion
	t.Run("BlockDeviceMappings_ToProto", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			BlockDeviceMappings: types.ListValueMust(
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
								"delete_on_termination": types.BoolType,
								"encrypted":             types.BoolType,
								"snapshot_id":           types.StringType,
							},
						},
					},
				},
				[]attr.Value{
					types.ObjectValueMust(
						map[string]attr.Type{
							"device_name": types.StringType,
							"ebs": types.ObjectType{
								AttrTypes: map[string]attr.Type{
									"volume_size":           types.StringType,
									"volume_type":           types.StringType,
									"iops":                  types.Int64Type,
									"throughput":            types.Int64Type,
									"kms_key_id":            types.StringType,
									"delete_on_termination": types.BoolType,
									"encrypted":             types.BoolType,
									"snapshot_id":           types.StringType,
								},
							},
						},
						map[string]attr.Value{
							"device_name": types.StringValue("/dev/xvda"),
							"ebs": types.ObjectValueMust(
								map[string]attr.Type{
									"volume_size":           types.StringType,
									"volume_type":           types.StringType,
									"iops":                  types.Int64Type,
									"throughput":            types.Int64Type,
									"kms_key_id":            types.StringType,
									"delete_on_termination": types.BoolType,
									"encrypted":             types.BoolType,
									"snapshot_id":           types.StringType,
								},
								map[string]attr.Value{
									"volume_size":           types.StringValue("100Gi"),
									"volume_type":           types.StringValue("gp3"),
									"iops":                  types.Int64Value(3000),
									"throughput":            types.Int64Value(125),
									"kms_key_id":            types.StringValue("arn:aws:kms:us-east-1:123456789012:key/12345"),
									"delete_on_termination": types.BoolValue(true),
									"encrypted":             types.BoolValue(true),
									"snapshot_id":           types.StringValue("snap-12345"),
								},
							),
						},
					),
				},
			),
			AmiFamily:                  types.StringValue("AL2"),
			SubnetSelectorTerms:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:           types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}

		// Test toProto
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.BlockDeviceMappings) != 1 {
			t.Fatalf("Expected 1 block device mapping, got %d", len(proto.BlockDeviceMappings))
		}
		bdm := proto.BlockDeviceMappings[0]
		if bdm.DeviceName == nil || *bdm.DeviceName != "/dev/xvda" {
			t.Errorf("Expected device name '/dev/xvda', got %v", bdm.DeviceName)
		}
		if bdm.Ebs == nil {
			t.Fatal("Expected non-nil EBS")
		}
		if bdm.Ebs.VolumeSize == nil || *bdm.Ebs.VolumeSize != "100Gi" {
			t.Errorf("Expected volume size '100Gi', got %v", bdm.Ebs.VolumeSize)
		}
		if bdm.Ebs.VolumeType == nil || *bdm.Ebs.VolumeType != "gp3" {
			t.Errorf("Expected volume type 'gp3', got %v", bdm.Ebs.VolumeType)
		}
		if bdm.Ebs.Iops == nil || *bdm.Ebs.Iops != 3000 {
			t.Errorf("Expected iops 3000, got %v", bdm.Ebs.Iops)
		}
		if bdm.Ebs.Throughput == nil || *bdm.Ebs.Throughput != 125 {
			t.Errorf("Expected throughput 125, got %v", bdm.Ebs.Throughput)
		}
		if bdm.Ebs.KmsKeyId == nil || *bdm.Ebs.KmsKeyId != "arn:aws:kms:us-east-1:123456789012:key/12345" {
			t.Errorf("Expected kms_key_id 'arn:aws:kms:us-east-1:123456789012:key/12345', got %v", bdm.Ebs.KmsKeyId)
		}
		if bdm.Ebs.DeleteOnTermination == nil || !*bdm.Ebs.DeleteOnTermination {
			t.Error("Expected delete_on_termination to be true")
		}
		if bdm.Ebs.Encrypted == nil || !*bdm.Ebs.Encrypted {
			t.Error("Expected encrypted to be true")
		}
		if bdm.Ebs.SnapshotId == nil || *bdm.Ebs.SnapshotId != "snap-12345" {
			t.Errorf("Expected snapshot_id 'snap-12345', got %v", bdm.Ebs.SnapshotId)
		}
	})
}

func validateNodePolicySchema(t *testing.T, schema schema.Schema) {
	// Validate required attributes
	requiredAttrs := []string{"name"}
	for _, attr := range requiredAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Required attribute %s not found in schema", attr)
		}
	}

	// Validate computed attributes
	computedAttrs := []string{"id"}
	for _, attr := range computedAttrs {
		if attrSchema, exists := schema.Attributes[attr]; exists {
			if !attrSchema.IsComputed() {
				t.Errorf("Attribute %s should be computed", attr)
			}
		}
	}

	// Validate optional attributes exist
	optionalAttrs := []string{
		"description", "weight",
		"instance_categories", "instance_families", "instance_cpus",
		"instance_hypervisors", "instance_generations", "instance_sizes",
		"zones", "architectures", "capacity_types", "operating_systems",
		"labels", "taints", "disruption", "limits",
		"node_pool_name", "node_class_name",
		"aws", "azure", "raw",
	}
	for _, attr := range optionalAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Optional attribute %s not found in schema", attr)
		}
	}

	// Validate tooltip fields exist
	tooltipAttrs := []string{
		"instance_categories_tip", "instance_families_tip", "instance_cpus_tip",
		"zones_tip", "architectures_tip", "capacity_type_tip", "operating_systems_tip",
		"taints_tip", "disruptions_tip", "limits_tip",
	}
	for _, attr := range tooltipAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Tooltip attribute %s not found in schema", attr)
		}
	}

	// Validate nested attributes exist (simplified validation)
	if _, exists := schema.Attributes["aws"]; !exists {
		t.Error("AWS configuration not found in schema")
	}

	if _, exists := schema.Attributes["azure"]; !exists {
		t.Error("Azure configuration not found in schema")
	}
}
