package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
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

	// Test LabelSelector with multiple MatchExpressions and multiple values
	t.Run("LabelSelector_WithMultipleMatchExpressions", func(t *testing.T) {
		selector := &LabelSelector{
			MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"app": types.StringValue("api"),
				"env": types.StringValue("prod"),
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
					types.ObjectValueMust(
						map[string]attr.Type{
							"key":      types.StringType,
							"operator": types.StringType,
							"values":   types.ListType{ElemType: types.StringType},
						},
						map[string]attr.Value{
							"key":      types.StringValue("instanceFamily"),
							"operator": types.StringValue("In"),
							"values": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("m5"),
								types.StringValue("m6i"),
								types.StringValue("m7i"),
							}),
						},
					),
					types.ObjectValueMust(
						map[string]attr.Type{
							"key":      types.StringType,
							"operator": types.StringType,
							"values":   types.ListType{ElemType: types.StringType},
						},
						map[string]attr.Value{
							"key":      types.StringValue("zone"),
							"operator": types.StringValue("NotIn"),
							"values": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("us-west-2a"),
								types.StringValue("us-west-2b"),
							}),
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
		if len(proto.MatchLabels) != 2 {
			t.Errorf("Expected 2 match labels, got %d", len(proto.MatchLabels))
		}
		if proto.MatchLabels["app"] != "api" {
			t.Errorf("Expected app=api, got %s", proto.MatchLabels["app"])
		}
		if proto.MatchLabels["env"] != "prod" {
			t.Errorf("Expected env=prod, got %s", proto.MatchLabels["env"])
		}
		if len(proto.MatchExpressions) != 3 {
			t.Fatalf("Expected 3 match expressions, got %d", len(proto.MatchExpressions))
		}

		// Check first expression (Gt with single value)
		expr1 := proto.MatchExpressions[0]
		if expr1.Key != "instanceGenerations" {
			t.Errorf("Expected key=instanceGenerations, got %s", expr1.Key)
		}
		if expr1.Operator != 5 { // GT = 5
			t.Errorf("Expected operator=GT (5), got %d", expr1.Operator)
		}
		if len(expr1.Values) != 1 {
			t.Errorf("Expected 1 value, got %d", len(expr1.Values))
		}
		if expr1.Values[0] != "4" {
			t.Errorf("Expected value '4', got %s", expr1.Values[0])
		}

		// Check second expression (In with multiple values)
		expr2 := proto.MatchExpressions[1]
		if expr2.Key != "instanceFamily" {
			t.Errorf("Expected key=instanceFamily, got %s", expr2.Key)
		}
		if expr2.Operator != 1 { // IN = 1
			t.Errorf("Expected operator=IN (1), got %d", expr2.Operator)
		}
		if len(expr2.Values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(expr2.Values))
		}
		expectedValues := []string{"m5", "m6i", "m7i"}
		for i, expected := range expectedValues {
			if expr2.Values[i] != expected {
				t.Errorf("Expected value[%d]='%s', got '%s'", i, expected, expr2.Values[i])
			}
		}

		// Check third expression (NotIn with multiple values)
		expr3 := proto.MatchExpressions[2]
		if expr3.Key != "zone" {
			t.Errorf("Expected key=zone, got %s", expr3.Key)
		}
		if expr3.Operator != 2 { // NOT_IN = 2
			t.Errorf("Expected operator=NOT_IN (2), got %d", expr3.Operator)
		}
		if len(expr3.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(expr3.Values))
		}
		expectedZones := []string{"us-west-2a", "us-west-2b"}
		for i, expected := range expectedZones {
			if expr3.Values[i] != expected {
				t.Errorf("Expected value[%d]='%s', got '%s'", i, expected, expr3.Values[i])
			}
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

	// Test AWS kubelet configuration
	t.Run("AWSKubelet_ToProto", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			SubnetSelectorTerms:              types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms:       types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			CapacityReservationSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:                 types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings:              types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			Kubelet: &KubeletConfiguration{
				MaxPods:                     types.Int32Value(110),
				PodsPerCore:                 types.Int32Value(10),
				CpuCfsQuota:                 types.BoolValue(true),
				ClusterDns:                  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.96.0.10")}),
				SystemReserved:              types.MapNull(types.StringType),
				KubeReserved:                types.MapNull(types.StringType),
				EvictionHard:                types.MapNull(types.StringType),
				EvictionSoft:                types.MapNull(types.StringType),
				EvictionSoftGracePeriod:     types.MapNull(types.StringType),
				EvictionMaxPodGracePeriod:   types.Int32Null(),
				ImageGcHighThresholdPercent: types.Int32Value(85),
				ImageGcLowThresholdPercent:  types.Int32Value(70),
			},
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto.Kubelet == nil {
			t.Fatal("Expected non-nil Kubelet")
		}
		if proto.Kubelet.MaxPods == nil || *proto.Kubelet.MaxPods != 110 {
			t.Errorf("Expected MaxPods=110, got %v", proto.Kubelet.MaxPods)
		}
		if proto.Kubelet.ImageGcHighThresholdPercent == nil || *proto.Kubelet.ImageGcHighThresholdPercent != 85 {
			t.Errorf("Expected ImageGcHighThresholdPercent=85, got %v", proto.Kubelet.ImageGcHighThresholdPercent)
		}
		if len(proto.Kubelet.ClusterDns) != 1 || proto.Kubelet.ClusterDns[0] != "10.96.0.10" {
			t.Errorf("Expected ClusterDns=[10.96.0.10], got %v", proto.Kubelet.ClusterDns)
		}
	})

	// Test AWS kubelet from proto
	t.Run("AWSKubelet_FromProto", func(t *testing.T) {
		maxPods := int32(110)
		high := int32(85)
		low := int32(70)
		proto := &apiv1.AWSNodeClassSpec{
			Kubelet: &apiv1.KubeletConfiguration{
				MaxPods:                     &maxPods,
				ImageGcHighThresholdPercent: &high,
				ImageGcLowThresholdPercent:  &low,
				ClusterDns:                  []string{"10.96.0.10"},
			},
		}
		aws := awsNodeClassFromProto(proto)
		if aws.Kubelet == nil {
			t.Fatal("Expected non-nil Kubelet")
		}
		if aws.Kubelet.MaxPods.ValueInt32() != 110 {
			t.Errorf("Expected MaxPods=110, got %d", aws.Kubelet.MaxPods.ValueInt32())
		}
		if aws.Kubelet.ImageGcHighThresholdPercent.ValueInt32() != 85 {
			t.Errorf("Expected ImageGcHighThresholdPercent=85, got %d", aws.Kubelet.ImageGcHighThresholdPercent.ValueInt32())
		}
		clusterDnsElems := aws.Kubelet.ClusterDns.Elements()
		if len(clusterDnsElems) != 1 {
			t.Fatalf("Expected 1 DNS entry, got %d", len(clusterDnsElems))
		}
	})

	// Test AWS context
	t.Run("AWSContext_ToProto", func(t *testing.T) {
		awsConfig := &AWSNodeClass{
			SubnetSelectorTerms:              types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms:       types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			CapacityReservationSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:                 types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings:              types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			Context:                          types.StringValue("arn:aws:ec2:us-east-1:123456789012:launch-template/lt-1234"),
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto.Context == nil || *proto.Context != "arn:aws:ec2:us-east-1:123456789012:launch-template/lt-1234" {
			t.Errorf("Expected Context ARN, got %v", proto.Context)
		}
	})

	// Test Azure kubelet
	t.Run("AzureKubelet_ToProto", func(t *testing.T) {
		azureConfig := &AzureNodeClass{
			Kubelet: &AzureKubeletConfiguration{
				CpuManagerPolicy:            types.StringValue("static"),
				CpuCfsQuota:                 types.BoolValue(true),
				CpuCfsQuotaPeriod:           types.StringValue("100ms"),
				ImageGcHighThresholdPercent: types.Int32Value(85),
				ImageGcLowThresholdPercent:  types.Int32Value(70),
				TopologyManagerPolicy:       types.StringValue("restricted"),
				AllowedUnsafeSysctls:        types.ListValueMust(types.StringType, []attr.Value{types.StringValue("net.ipv4.tcp_syncookies")}),
				ContainerLogMaxSize:         types.StringValue("50Mi"),
				ContainerLogMaxFiles:        types.Int32Value(5),
				PodPidsLimit:                types.Int64Value(4096),
			},
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := azureConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto.Kubelet == nil {
			t.Fatal("Expected non-nil Kubelet")
		}
		if proto.Kubelet.CpuManagerPolicy == nil || *proto.Kubelet.CpuManagerPolicy != "static" {
			t.Errorf("Expected CpuManagerPolicy=static, got %v", proto.Kubelet.CpuManagerPolicy)
		}
		if proto.Kubelet.CpuCfsQuota == nil || !*proto.Kubelet.CpuCfsQuota {
			t.Error("Expected CpuCfsQuota=true")
		}
		if proto.Kubelet.ContainerLogMaxSize == nil || *proto.Kubelet.ContainerLogMaxSize != "50Mi" {
			t.Errorf("Expected ContainerLogMaxSize=50Mi, got %v", proto.Kubelet.ContainerLogMaxSize)
		}
		if proto.Kubelet.PodPidsLimit == nil || *proto.Kubelet.PodPidsLimit != 4096 {
			t.Errorf("Expected PodPidsLimit=4096, got %v", proto.Kubelet.PodPidsLimit)
		}
		if len(proto.Kubelet.AllowedUnsafeSysctls) != 1 {
			t.Errorf("Expected 1 sysctl, got %d", len(proto.Kubelet.AllowedUnsafeSysctls))
		}
	})

	// Test Azure kubelet from proto
	t.Run("AzureKubelet_FromProto", func(t *testing.T) {
		policy := "static"
		high := int32(85)
		low := int32(70)
		logSize := "50Mi"
		logFiles := int32(5)
		podPids := int64(4096)
		proto := &apiv1.AzureNodeClassSpec{
			Kubelet: &apiv1.AzureKubeletConfiguration{
				CpuManagerPolicy:            &policy,
				ImageGcHighThresholdPercent: &high,
				ImageGcLowThresholdPercent:  &low,
				ContainerLogMaxSize:         &logSize,
				ContainerLogMaxFiles:        &logFiles,
				PodPidsLimit:                &podPids,
				AllowedUnsafeSysctls:        []string{"net.ipv4.tcp_syncookies"},
			},
		}
		azure := azureNodeClassFromProto(proto)
		if azure.Kubelet == nil {
			t.Fatal("Expected non-nil Kubelet")
		}
		if azure.Kubelet.CpuManagerPolicy.ValueString() != "static" {
			t.Errorf("Expected CpuManagerPolicy=static, got %s", azure.Kubelet.CpuManagerPolicy.ValueString())
		}
		if azure.Kubelet.ImageGcHighThresholdPercent.ValueInt32() != 85 {
			t.Errorf("Expected ImageGcHighThresholdPercent=85, got %d", azure.Kubelet.ImageGcHighThresholdPercent.ValueInt32())
		}
		if azure.Kubelet.PodPidsLimit.ValueInt64() != 4096 {
			t.Errorf("Expected PodPidsLimit=4096, got %d", azure.Kubelet.PodPidsLimit.ValueInt64())
		}
		sysctls := azure.Kubelet.AllowedUnsafeSysctls.Elements()
		if len(sysctls) != 1 {
			t.Fatalf("Expected 1 sysctl, got %d", len(sysctls))
		}
	})

	// Test instance_types via LabelSelector conversion
	t.Run("InstanceTypes_LabelSelector_ToProto", func(t *testing.T) {
		ctx := context.Background()
		sel := &LabelSelector{
			MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"karpenter.k8s.aws/instance-type": types.StringValue("m5.xlarge"),
			}),
			MatchExpressions: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
				"key":      types.StringType,
				"operator": types.StringType,
				"values":   types.ListType{ElemType: types.StringType},
			}}),
		}
		proto, err := sel.toProto(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if proto.MatchLabels["karpenter.k8s.aws/instance-type"] != "m5.xlarge" {
			t.Errorf("Expected match_labels to contain instance type, got %v", proto.MatchLabels)
		}
	})

	// Test instance_types via LabelSelector fromProto
	t.Run("InstanceTypes_LabelSelector_FromProto", func(t *testing.T) {
		proto := &apiv1.LabelSelector{
			MatchLabels: map[string]string{"karpenter.k8s.aws/instance-type": "m5.xlarge"},
		}
		sel := labelSelectorFromProto(proto)
		if sel == nil {
			t.Fatal("Expected non-nil selector")
		}
		if sel.MatchLabels.IsNull() {
			t.Fatal("Expected non-null MatchLabels")
		}
		elems := sel.MatchLabels.Elements()
		if len(elems) != 1 {
			t.Errorf("Expected 1 label, got %d", len(elems))
		}
	})

	// Test AmiSelectorTerms ssm_parameter field
	t.Run("AmiSelectorTerms_SsmParameter_ToProto", func(t *testing.T) {
		attrTypes := map[string]attr.Type{
			"tags":          types.MapType{ElemType: types.StringType},
			"id":            types.StringType,
			"name":          types.StringType,
			"owner":         types.StringType,
			"alias":         types.StringType,
			"ssm_parameter": types.StringType,
		}
		awsConfig := &AWSNodeClass{
			AmiSelectorTerms: types.ListValueMust(
				types.ObjectType{AttrTypes: attrTypes},
				[]attr.Value{
					types.ObjectValueMust(attrTypes, map[string]attr.Value{
						"tags":          types.MapNull(types.StringType),
						"id":            types.StringNull(),
						"name":          types.StringNull(),
						"owner":         types.StringNull(),
						"alias":         types.StringNull(),
						"ssm_parameter": types.StringValue("/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id"),
					}),
				},
			),
			SubnetSelectorTerms:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			BlockDeviceMappings:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if len(proto.AmiSelectorTerms) != 1 {
			t.Fatalf("Expected 1 AMI selector term, got %d", len(proto.AmiSelectorTerms))
		}
		if proto.AmiSelectorTerms[0].SsmParameter != "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id" {
			t.Errorf("Expected ssm_parameter to be set, got %q", proto.AmiSelectorTerms[0].SsmParameter)
		}
	})

	t.Run("AmiSelectorTerms_SsmParameter_FromProto", func(t *testing.T) {
		proto := &apiv1.AWSNodeClassSpec{
			AmiSelectorTerms: []*apiv1.AMISelectorTerm{
				{SsmParameter: "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id"},
			},
		}
		aws := awsNodeClassFromProto(proto)
		elems := aws.AmiSelectorTerms.Elements()
		if len(elems) != 1 {
			t.Fatalf("Expected 1 AMI selector term, got %d", len(elems))
		}
		obj, ok := elems[0].(types.Object)
		if !ok {
			t.Fatal("Expected object element")
		}
		ssmParam, ok := obj.Attributes()["ssm_parameter"].(types.String)
		if !ok || ssmParam.ValueString() != "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id" {
			t.Errorf("Expected ssm_parameter to round-trip, got %v", obj.Attributes()["ssm_parameter"])
		}
	})

	// Test BlockDeviceMappings root_volume and Ebs volume_initialization_rate
	t.Run("BlockDeviceMappings_RootVolumeAndInitRate_ToProto", func(t *testing.T) {
		ebsAttrTypes := map[string]attr.Type{
			"volume_size":                types.StringType,
			"volume_type":                types.StringType,
			"iops":                       types.Int64Type,
			"throughput":                 types.Int64Type,
			"kms_key_id":                 types.StringType,
			"delete_on_termination":      types.BoolType,
			"encrypted":                  types.BoolType,
			"snapshot_id":                types.StringType,
			"volume_initialization_rate": types.Int32Type,
		}
		mappingAttrTypes := map[string]attr.Type{
			"device_name": types.StringType,
			"root_volume": types.BoolType,
			"ebs":         types.ObjectType{AttrTypes: ebsAttrTypes},
		}
		awsConfig := &AWSNodeClass{
			BlockDeviceMappings: types.ListValueMust(
				types.ObjectType{AttrTypes: mappingAttrTypes},
				[]attr.Value{
					types.ObjectValueMust(mappingAttrTypes, map[string]attr.Value{
						"device_name": types.StringValue("/dev/xvda"),
						"root_volume": types.BoolValue(true),
						"ebs": types.ObjectValueMust(ebsAttrTypes, map[string]attr.Value{
							"volume_size":                types.StringValue("100Gi"),
							"volume_type":                types.StringValue("gp3"),
							"iops":                       types.Int64Null(),
							"throughput":                 types.Int64Null(),
							"kms_key_id":                 types.StringNull(),
							"delete_on_termination":      types.BoolNull(),
							"encrypted":                  types.BoolNull(),
							"snapshot_id":                types.StringNull(),
							"volume_initialization_rate": types.Int32Value(50),
						}),
					}),
				},
			),
			AmiFamily:                  types.StringValue("AL2"),
			SubnetSelectorTerms:        types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			SecurityGroupSelectorTerms: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
			AmiSelectorTerms:           types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}),
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := awsConfig.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if len(proto.BlockDeviceMappings) != 1 {
			t.Fatalf("Expected 1 block device mapping, got %d", len(proto.BlockDeviceMappings))
		}
		bdm := proto.BlockDeviceMappings[0]
		if bdm.RootVolume == nil || !*bdm.RootVolume {
			t.Error("Expected root_volume to be true")
		}
		if bdm.Ebs == nil || bdm.Ebs.VolumeInitializationRate == nil || *bdm.Ebs.VolumeInitializationRate != 50 {
			t.Errorf("Expected volume_initialization_rate=50, got %v", bdm.Ebs.VolumeInitializationRate)
		}
	})

	t.Run("BlockDeviceMappings_RootVolumeAndInitRate_FromProto", func(t *testing.T) {
		rootVolume := true
		initRate := int32(50)
		proto := &apiv1.AWSNodeClassSpec{
			BlockDeviceMappings: []*apiv1.BlockDeviceMapping{
				{
					RootVolume: &rootVolume,
					Ebs: &apiv1.BlockDevice{
						VolumeInitializationRate: &initRate,
					},
				},
			},
		}
		aws := awsNodeClassFromProto(proto)
		elems := aws.BlockDeviceMappings.Elements()
		if len(elems) != 1 {
			t.Fatalf("Expected 1 block device mapping, got %d", len(elems))
		}
		obj, ok := elems[0].(types.Object)
		if !ok {
			t.Fatal("Expected object element")
		}
		rootVol, ok := obj.Attributes()["root_volume"].(types.Bool)
		if !ok || !rootVol.ValueBool() {
			t.Errorf("Expected root_volume=true, got %v", obj.Attributes()["root_volume"])
		}
		ebsObj, ok := obj.Attributes()["ebs"].(types.Object)
		if !ok {
			t.Fatal("Expected ebs object")
		}
		rate, ok := ebsObj.Attributes()["volume_initialization_rate"].(types.Int32)
		if !ok || rate.ValueInt32() != 50 {
			t.Errorf("Expected volume_initialization_rate=50, got %v", ebsObj.Attributes()["volume_initialization_rate"])
		}
	})

	// Test top-level InstanceShapes, InstanceShapesTip, InstanceLocalNvmeTip, StartupTaintsTip
	t.Run("NodePolicy_NewTopLevelFields_ToProto", func(t *testing.T) {
		model := &NodePolicyResourceModel{
			Name: types.StringValue("test-policy"),
			InstanceShapes: &LabelSelector{
				MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"karpenter.k8s.aws/instance-shape": types.StringValue("standard"),
				}),
				MatchExpressions: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
					"key": types.StringType, "operator": types.StringType, "values": types.ListType{ElemType: types.StringType},
				}}),
			},
			InstanceShapesTip:    types.StringValue("Select instance shapes"),
			InstanceLocalNvmeTip: types.StringValue("Local NVMe tip"),
			StartupTaintsTip:     types.StringValue("Startup taints tip"),
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := model.toProto(ctx, &diags, "test-team-id")
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto.InstanceShapes == nil {
			t.Fatal("Expected non-nil InstanceShapes")
		}
		if proto.InstanceShapes.MatchLabels["karpenter.k8s.aws/instance-shape"] != "standard" {
			t.Errorf("Expected instance shape label, got %v", proto.InstanceShapes.MatchLabels)
		}
		if proto.InstanceShapesTip == nil || *proto.InstanceShapesTip != "Select instance shapes" {
			t.Errorf("Expected InstanceShapesTip, got %v", proto.InstanceShapesTip)
		}
		if proto.InstanceLocalNvmeTip == nil || *proto.InstanceLocalNvmeTip != "Local NVMe tip" {
			t.Errorf("Expected InstanceLocalNvmeTip, got %v", proto.InstanceLocalNvmeTip)
		}
		if proto.StartupTaintsTip == nil || *proto.StartupTaintsTip != "Startup taints tip" {
			t.Errorf("Expected StartupTaintsTip, got %v", proto.StartupTaintsTip)
		}
	})

	// Test GCP node class conversion
	t.Run("GCPNodeClass_ToProto", func(t *testing.T) {
		imageSelectorAttrTypes := map[string]attr.Type{"alias": types.StringType, "id": types.StringType}
		diskAttrTypes := map[string]attr.Type{
			"size_gib":             types.Int32Type,
			"category":             types.StringType,
			"boot":                 types.BoolType,
			"secondary_boot_image": types.StringType,
			"secondary_boot_mode":  types.StringType,
		}
		gcp := &GCPNodeClass{
			ServiceAccount: types.StringValue("my-service-account@project.iam.gserviceaccount.com"),
			ImageSelectorTerms: types.ListValueMust(
				types.ObjectType{AttrTypes: imageSelectorAttrTypes},
				[]attr.Value{
					types.ObjectValueMust(imageSelectorAttrTypes, map[string]attr.Value{
						"alias": types.StringValue("ubuntu"),
						"id":    types.StringNull(),
					}),
				},
			),
			ImageFamily: types.StringValue("ubuntu-2204-lts"),
			Kubelet: &KubeletConfiguration{
				MaxPods:                     types.Int32Value(110),
				PodsPerCore:                 types.Int32Null(),
				CpuCfsQuota:                 types.BoolNull(),
				ClusterDns:                  types.ListNull(types.StringType),
				SystemReserved:              types.MapNull(types.StringType),
				KubeReserved:                types.MapNull(types.StringType),
				EvictionHard:                types.MapNull(types.StringType),
				EvictionSoft:                types.MapNull(types.StringType),
				EvictionSoftGracePeriod:     types.MapNull(types.StringType),
				EvictionMaxPodGracePeriod:   types.Int32Null(),
				ImageGcHighThresholdPercent: types.Int32Null(),
				ImageGcLowThresholdPercent:  types.Int32Null(),
			},
			Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"env": types.StringValue("prod")}),
			Metadata:    types.MapNull(types.StringType),
			NetworkTags: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("allow-ssh")}),
			Disks: types.ListValueMust(
				types.ObjectType{AttrTypes: diskAttrTypes},
				[]attr.Value{
					types.ObjectValueMust(diskAttrTypes, map[string]attr.Value{
						"size_gib":             types.Int32Value(100),
						"category":             types.StringValue("pd-ssd"),
						"boot":                 types.BoolValue(true),
						"secondary_boot_image": types.StringValue(""),
						"secondary_boot_mode":  types.StringValue(""),
					}),
				},
			),
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := gcp.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto.ServiceAccount != "my-service-account@project.iam.gserviceaccount.com" {
			t.Errorf("Expected ServiceAccount to match, got %s", proto.ServiceAccount)
		}
		if len(proto.ImageSelectorTerms) != 1 || proto.ImageSelectorTerms[0].Alias != "ubuntu" {
			t.Errorf("Expected 1 image selector term with alias=ubuntu, got %v", proto.ImageSelectorTerms)
		}
		if proto.ImageFamily == nil || *proto.ImageFamily != "ubuntu-2204-lts" {
			t.Errorf("Expected ImageFamily=ubuntu-2204-lts, got %v", proto.ImageFamily)
		}
		if proto.KubeletConfiguration == nil || proto.KubeletConfiguration.MaxPods == nil || *proto.KubeletConfiguration.MaxPods != 110 {
			t.Errorf("Expected KubeletConfiguration.MaxPods=110, got %v", proto.KubeletConfiguration)
		}
		if proto.Labels["env"] != "prod" {
			t.Errorf("Expected Labels[env]=prod, got %v", proto.Labels)
		}
		if len(proto.NetworkTags) != 1 || proto.NetworkTags[0] != "allow-ssh" {
			t.Errorf("Expected NetworkTags=[allow-ssh], got %v", proto.NetworkTags)
		}
		if len(proto.Disks) != 1 || proto.Disks[0].SizeGib != 100 || proto.Disks[0].Category != "pd-ssd" || !proto.Disks[0].Boot {
			t.Errorf("Expected 1 disk with size_gib=100 category=pd-ssd boot=true, got %v", proto.Disks)
		}
	})

	t.Run("GCPNodeClass_FromProto", func(t *testing.T) {
		imageFamily := "ubuntu-2204-lts"
		maxPods := int32(110)
		proto := &apiv1.GCPNodeClassSpec{
			ServiceAccount: "my-service-account@project.iam.gserviceaccount.com",
			ImageSelectorTerms: []*apiv1.GCPImageSelectorTerm{
				{Alias: "ubuntu"},
			},
			ImageFamily:          &imageFamily,
			KubeletConfiguration: &apiv1.KubeletConfiguration{MaxPods: &maxPods},
			Labels:               map[string]string{"env": "prod"},
			NetworkTags:          []string{"allow-ssh"},
			Disks: []*apiv1.GCPDisk{
				{SizeGib: 100, Category: "pd-ssd", Boot: true},
			},
		}
		gcp := gcpNodeClassFromProto(proto)
		if gcp.ServiceAccount.ValueString() != "my-service-account@project.iam.gserviceaccount.com" {
			t.Errorf("Expected ServiceAccount to round-trip, got %s", gcp.ServiceAccount.ValueString())
		}
		terms := gcp.ImageSelectorTerms.Elements()
		if len(terms) != 1 {
			t.Fatalf("Expected 1 image selector term, got %d", len(terms))
		}
		if gcp.ImageFamily.ValueString() != "ubuntu-2204-lts" {
			t.Errorf("Expected ImageFamily to round-trip, got %s", gcp.ImageFamily.ValueString())
		}
		if gcp.Kubelet == nil || gcp.Kubelet.MaxPods.ValueInt32() != 110 {
			t.Errorf("Expected Kubelet.MaxPods=110, got %v", gcp.Kubelet)
		}
		if gcp.Labels.IsNull() || len(gcp.Labels.Elements()) != 1 {
			t.Errorf("Expected 1 label, got %v", gcp.Labels)
		}
		disks := gcp.Disks.Elements()
		if len(disks) != 1 {
			t.Fatalf("Expected 1 disk, got %d", len(disks))
		}
	})

	t.Run("GCPSpecEmpty", func(t *testing.T) {
		if !isGCPSpecEmpty(nil) {
			t.Error("Expected nil spec to be empty")
		}
		if !isGCPSpecEmpty(&apiv1.GCPNodeClassSpec{}) {
			t.Error("Expected zero-value spec to be empty")
		}
		if isGCPSpecEmpty(&apiv1.GCPNodeClassSpec{ServiceAccount: "sa@project.iam.gserviceaccount.com"}) {
			t.Error("Expected spec with ServiceAccount to be non-empty")
		}
	})

	// Test OCI node class conversion
	t.Run("OCINodeClass_ToProto", func(t *testing.T) {
		imageSelectorAttrTypes := map[string]attr.Type{"id": types.StringType, "name": types.StringType, "compartment_id": types.StringType}
		idNameAttrTypes := map[string]attr.Type{"id": types.StringType, "name": types.StringType}
		blockDeviceAttrTypes := map[string]attr.Type{"size_in_gbs": types.Int64Type, "vpus_per_gb": types.Int64Type}
		oci := &OCINodeClass{
			VcnId: types.StringValue("ocid1.vcn.oc1..aaaa"),
			ImageSelector: types.ListValueMust(
				types.ObjectType{AttrTypes: imageSelectorAttrTypes},
				[]attr.Value{
					types.ObjectValueMust(imageSelectorAttrTypes, map[string]attr.Value{
						"id":             types.StringValue("ocid1.image.oc1..bbbb"),
						"name":           types.StringNull(),
						"compartment_id": types.StringNull(),
					}),
				},
			),
			SubnetSelector: types.ListValueMust(
				types.ObjectType{AttrTypes: idNameAttrTypes},
				[]attr.Value{
					types.ObjectValueMust(idNameAttrTypes, map[string]attr.Value{
						"id": types.StringValue("ocid1.subnet.oc1..cccc"), "name": types.StringNull(),
					}),
				},
			),
			SecurityGroupSelector: types.ListNull(types.ObjectType{AttrTypes: idNameAttrTypes}),
			UserData:              types.StringValue("#!/bin/bash\necho hi"),
			PreInstallScript:      types.StringNull(),
			MetaData:              types.MapNull(types.StringType),
			ImageFamily:           types.StringValue("oracle-linux-8"),
			Tags:                  types.MapNull(types.StringType),
			FreeFormTags:          types.MapValueMust(types.StringType, map[string]attr.Value{"env": types.StringValue("prod")}),
			BootConfig: &OCIBootConfig{
				BootVolumeSizeInGbs: types.Int64Value(100),
				BootVolumeVpusPerGb: types.Int64Value(10),
			},
			LaunchOptions: &OCILaunchOptions{
				BootVolumeType:                  types.StringValue("PARAVIRTUALIZED"),
				Firmware:                        types.StringNull(),
				NetworkType:                     types.StringNull(),
				RemoteDataVolumeType:            types.StringNull(),
				IsConsistentVolumeNamingEnabled: types.BoolValue(true),
			},
			BlockDevices: types.ListValueMust(
				types.ObjectType{AttrTypes: blockDeviceAttrTypes},
				[]attr.Value{
					types.ObjectValueMust(blockDeviceAttrTypes, map[string]attr.Value{
						"size_in_gbs": types.Int64Value(50),
						"vpus_per_gb": types.Int64Value(10),
					}),
				},
			),
			AgentList: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("bastion")}),
		}
		ctx := context.Background()
		var diags diag.Diagnostics
		proto := oci.toProto(ctx, &diags)
		if diags.HasError() {
			t.Fatalf("Expected no error, got %v", diags)
		}
		if proto.VcnId != "ocid1.vcn.oc1..aaaa" {
			t.Errorf("Expected VcnId to match, got %s", proto.VcnId)
		}
		if len(proto.ImageSelector) != 1 || proto.ImageSelector[0].Id != "ocid1.image.oc1..bbbb" {
			t.Errorf("Expected 1 image selector with id, got %v", proto.ImageSelector)
		}
		if len(proto.SubnetSelector) != 1 || proto.SubnetSelector[0].Id != "ocid1.subnet.oc1..cccc" {
			t.Errorf("Expected 1 subnet selector with id, got %v", proto.SubnetSelector)
		}
		if proto.UserData == nil || *proto.UserData != "#!/bin/bash\necho hi" {
			t.Errorf("Expected UserData to match, got %v", proto.UserData)
		}
		if proto.ImageFamily != "oracle-linux-8" {
			t.Errorf("Expected ImageFamily to match, got %s", proto.ImageFamily)
		}
		if proto.FreeFormTags["env"] != "prod" {
			t.Errorf("Expected FreeFormTags[env]=prod, got %v", proto.FreeFormTags)
		}
		if proto.BootConfig == nil || proto.BootConfig.BootVolumeSizeInGbs != 100 || proto.BootConfig.BootVolumeVpusPerGb != 10 {
			t.Errorf("Expected BootConfig with size=100 vpus=10, got %v", proto.BootConfig)
		}
		if proto.LaunchOptions == nil || proto.LaunchOptions.BootVolumeType == nil || *proto.LaunchOptions.BootVolumeType != "PARAVIRTUALIZED" {
			t.Errorf("Expected LaunchOptions.BootVolumeType=PARAVIRTUALIZED, got %v", proto.LaunchOptions)
		}
		if proto.LaunchOptions.IsConsistentVolumeNamingEnabled == nil || !*proto.LaunchOptions.IsConsistentVolumeNamingEnabled {
			t.Error("Expected IsConsistentVolumeNamingEnabled=true")
		}
		if len(proto.BlockDevices) != 1 || proto.BlockDevices[0].SizeInGbs != 50 || proto.BlockDevices[0].VpusPerGb != 10 {
			t.Errorf("Expected 1 block device with size=50 vpus=10, got %v", proto.BlockDevices)
		}
		if len(proto.AgentList) != 1 || proto.AgentList[0] != "bastion" {
			t.Errorf("Expected AgentList=[bastion], got %v", proto.AgentList)
		}
	})

	t.Run("OCINodeClass_FromProto", func(t *testing.T) {
		userData := "#!/bin/bash\necho hi"
		bootVolType := "PARAVIRTUALIZED"
		consistentNaming := true
		proto := &apiv1.OCINodeClassSpec{
			VcnId: "ocid1.vcn.oc1..aaaa",
			ImageSelector: []*apiv1.OCIImageSelectorTerm{
				{Id: "ocid1.image.oc1..bbbb"},
			},
			UserData:     &userData,
			ImageFamily:  "oracle-linux-8",
			FreeFormTags: map[string]string{"env": "prod"},
			BootConfig: &apiv1.OCIBootConfig{
				BootVolumeSizeInGbs: 100,
				BootVolumeVpusPerGb: 10,
			},
			LaunchOptions: &apiv1.OCILaunchOptions{
				BootVolumeType:                  &bootVolType,
				IsConsistentVolumeNamingEnabled: &consistentNaming,
			},
			BlockDevices: []*apiv1.OCIVolumeAttributes{
				{SizeInGbs: 50, VpusPerGb: 10},
			},
			AgentList: []string{"bastion"},
		}
		oci := ociNodeClassFromProto(proto)
		if oci.VcnId.ValueString() != "ocid1.vcn.oc1..aaaa" {
			t.Errorf("Expected VcnId to round-trip, got %s", oci.VcnId.ValueString())
		}
		if len(oci.ImageSelector.Elements()) != 1 {
			t.Errorf("Expected 1 image selector, got %d", len(oci.ImageSelector.Elements()))
		}
		if oci.UserData.ValueString() != userData {
			t.Errorf("Expected UserData to round-trip, got %s", oci.UserData.ValueString())
		}
		if oci.BootConfig == nil || oci.BootConfig.BootVolumeSizeInGbs.ValueInt64() != 100 {
			t.Errorf("Expected BootConfig to round-trip, got %v", oci.BootConfig)
		}
		if oci.LaunchOptions == nil || oci.LaunchOptions.BootVolumeType.ValueString() != "PARAVIRTUALIZED" {
			t.Errorf("Expected LaunchOptions to round-trip, got %v", oci.LaunchOptions)
		}
		if !oci.LaunchOptions.IsConsistentVolumeNamingEnabled.ValueBool() {
			t.Error("Expected IsConsistentVolumeNamingEnabled=true")
		}
		if len(oci.BlockDevices.Elements()) != 1 {
			t.Errorf("Expected 1 block device, got %d", len(oci.BlockDevices.Elements()))
		}
		if len(oci.AgentList.Elements()) != 1 {
			t.Errorf("Expected 1 agent, got %d", len(oci.AgentList.Elements()))
		}
	})

	t.Run("OCISpecEmpty", func(t *testing.T) {
		if !isOCISpecEmpty(nil) {
			t.Error("Expected nil spec to be empty")
		}
		if !isOCISpecEmpty(&apiv1.OCINodeClassSpec{}) {
			t.Error("Expected zero-value spec to be empty")
		}
		if isOCISpecEmpty(&apiv1.OCINodeClassSpec{VcnId: "ocid1.vcn.oc1..aaaa"}) {
			t.Error("Expected spec with VcnId to be non-empty")
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
		"instance_types", "instance_shapes",
		"zones", "architectures", "capacity_types", "operating_systems",
		"labels", "taints", "disruption", "limits",
		"node_pool_name", "node_class_name",
		"aws", "azure", "gcp", "oci", "raw",
	}
	for _, attr := range optionalAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Optional attribute %s not found in schema", attr)
		}
	}

	// Validate tooltip fields exist
	tooltipAttrs := []string{
		"instance_categories_tip", "instance_families_tip", "instance_cpus_tip",
		"instance_shapes_tip", "instance_local_nvme_tip", "startup_taints_tip",
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

	if _, exists := schema.Attributes["gcp"]; !exists {
		t.Error("GCP configuration not found in schema")
	}

	if _, exists := schema.Attributes["oci"]; !exists {
		t.Error("OCI configuration not found in schema")
	}
}
