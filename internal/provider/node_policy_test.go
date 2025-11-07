package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
