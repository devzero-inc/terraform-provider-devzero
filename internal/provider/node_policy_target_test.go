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

func TestNodePolicyTargetResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Instantiate the resource and call Schema
	nodePolicyTargetResource := NewNodePolicyTargetResource()
	nodePolicyTargetResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	// Validate the schema
	validateNodePolicyTargetSchema(t, resp.Schema)
}

func TestNodePolicyTargetResourceModel(t *testing.T) {
	t.Parallel()

	// Test NodePolicyTargetResourceModel
	t.Run("NodePolicyTargetResourceModel", func(t *testing.T) {
		model := &NodePolicyTargetResourceModel{
			Id:          types.StringValue("target-123"),
			PolicyId:    types.StringValue("policy-456"),
			Name:        types.StringValue("production-clusters"),
			Description: types.StringValue("Apply to all production clusters"),
			Enabled:     types.BoolValue(true),
			ClusterIds: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("cluster-1"),
				types.StringValue("cluster-2"),
				types.StringValue("cluster-3"),
			}),
		}

		if model.Id.ValueString() != "target-123" {
			t.Errorf("Expected id to be 'target-123', got %s", model.Id.ValueString())
		}
		if model.PolicyId.ValueString() != "policy-456" {
			t.Errorf("Expected policy_id to be 'policy-456', got %s", model.PolicyId.ValueString())
		}
		if model.Name.ValueString() != "production-clusters" {
			t.Errorf("Expected name to be 'production-clusters', got %s", model.Name.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Expected enabled to be true")
		}

		// Verify cluster IDs count
		clusterIds := model.ClusterIds.Elements()
		if len(clusterIds) != 3 {
			t.Errorf("Expected 3 cluster IDs, got %d", len(clusterIds))
		}
	})

	// Test toProto conversion
	t.Run("ToProto", func(t *testing.T) {
		model := &NodePolicyTargetResourceModel{
			Id:          types.StringValue("target-123"),
			PolicyId:    types.StringValue("policy-456"),
			Name:        types.StringValue("production-clusters"),
			Description: types.StringValue("Apply to all production clusters"),
			Enabled:     types.BoolValue(true),
			ClusterIds: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("cluster-1"),
				types.StringValue("cluster-2"),
			}),
		}

		ctx := context.Background()
		diags := diag.Diagnostics{}

		proto := model.toProto(ctx, &diags, "team-789")

		if diags.HasError() {
			t.Fatalf("toProto had errors: %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}

		// Validate proto fields
		if proto.TargetId != "target-123" {
			t.Errorf("Expected target_id to be 'target-123', got %s", proto.TargetId)
		}
		if proto.PolicyId != "policy-456" {
			t.Errorf("Expected policy_id to be 'policy-456', got %s", proto.PolicyId)
		}
		if proto.Name != "production-clusters" {
			t.Errorf("Expected name to be 'production-clusters', got %s", proto.Name)
		}
		if proto.Description != "Apply to all production clusters" {
			t.Errorf("Expected description, got %s", proto.Description)
		}
		if proto.TeamId != "team-789" {
			t.Errorf("Expected team_id to be 'team-789', got %s", proto.TeamId)
		}
		if !proto.Enabled {
			t.Error("Expected enabled to be true")
		}
		if len(proto.ClusterIds) != 2 {
			t.Errorf("Expected 2 cluster IDs in proto, got %d", len(proto.ClusterIds))
		}
		if proto.ClusterIds[0] != "cluster-1" {
			t.Errorf("Expected first cluster ID to be 'cluster-1', got %s", proto.ClusterIds[0])
		}
		if proto.ClusterIds[1] != "cluster-2" {
			t.Errorf("Expected second cluster ID to be 'cluster-2', got %s", proto.ClusterIds[1])
		}
	})

	// Test fromProto conversion
	t.Run("FromProto", func(t *testing.T) {
		proto := &apiv1.NodePolicyTarget{
			TargetId:    "target-123",
			PolicyId:    "policy-456",
			Name:        "production-clusters",
			Description: "Apply to all production clusters",
			TeamId:      "team-789",
			Enabled:     true,
			ClusterIds:  []string{"cluster-1", "cluster-2", "cluster-3"},
		}

		model := &NodePolicyTargetResourceModel{}
		model.fromProto(proto)

		// Validate model fields
		if model.Id.ValueString() != "target-123" {
			t.Errorf("Expected id to be 'target-123', got %s", model.Id.ValueString())
		}
		if model.PolicyId.ValueString() != "policy-456" {
			t.Errorf("Expected policy_id to be 'policy-456', got %s", model.PolicyId.ValueString())
		}
		if model.Name.ValueString() != "production-clusters" {
			t.Errorf("Expected name to be 'production-clusters', got %s", model.Name.ValueString())
		}
		if model.Description.ValueString() != "Apply to all production clusters" {
			t.Errorf("Expected description, got %s", model.Description.ValueString())
		}
		if !model.Enabled.ValueBool() {
			t.Error("Expected enabled to be true")
		}

		// Verify cluster IDs
		clusterIds := model.ClusterIds.Elements()
		if len(clusterIds) != 3 {
			t.Errorf("Expected 3 cluster IDs, got %d", len(clusterIds))
		}

		// Validate cluster ID values
		ctx := context.Background()
		var clusterIdStrings []string
		model.ClusterIds.ElementsAs(ctx, &clusterIdStrings, false)
		if len(clusterIdStrings) != 3 {
			t.Errorf("Expected 3 cluster IDs after conversion, got %d", len(clusterIdStrings))
		}
		if clusterIdStrings[0] != "cluster-1" {
			t.Errorf("Expected first cluster ID to be 'cluster-1', got %s", clusterIdStrings[0])
		}
		if clusterIdStrings[1] != "cluster-2" {
			t.Errorf("Expected second cluster ID to be 'cluster-2', got %s", clusterIdStrings[1])
		}
		if clusterIdStrings[2] != "cluster-3" {
			t.Errorf("Expected third cluster ID to be 'cluster-3', got %s", clusterIdStrings[2])
		}
	})

	// Test empty cluster IDs
	t.Run("EmptyClusterIds", func(t *testing.T) {
		model := &NodePolicyTargetResourceModel{
			Id:          types.StringValue("target-123"),
			PolicyId:    types.StringValue("policy-456"),
			Name:        types.StringValue("test"),
			Description: types.StringValue("test"),
			Enabled:     types.BoolValue(true),
			ClusterIds:  types.ListValueMust(types.StringType, []attr.Value{}),
		}

		ctx := context.Background()
		diags := diag.Diagnostics{}

		proto := model.toProto(ctx, &diags, "team-789")

		if diags.HasError() {
			t.Fatalf("toProto had errors: %v", diags)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if len(proto.ClusterIds) != 0 {
			t.Errorf("Expected 0 cluster IDs, got %d", len(proto.ClusterIds))
		}
	})

	// Test disabled target
	t.Run("DisabledTarget", func(t *testing.T) {
		model := &NodePolicyTargetResourceModel{
			Id:          types.StringValue("target-123"),
			PolicyId:    types.StringValue("policy-456"),
			Name:        types.StringValue("disabled-target"),
			Description: types.StringValue("This target is disabled"),
			Enabled:     types.BoolValue(false),
			ClusterIds: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("cluster-1"),
			}),
		}

		ctx := context.Background()
		diags := diag.Diagnostics{}

		proto := model.toProto(ctx, &diags, "team-789")

		if diags.HasError() {
			t.Fatalf("toProto had errors: %v", diags)
		}
		if proto.Enabled {
			t.Error("Expected enabled to be false")
		}
	})

	// Test round-trip conversion
	t.Run("RoundTrip", func(t *testing.T) {
		original := &apiv1.NodePolicyTarget{
			TargetId:    "target-xyz",
			PolicyId:    "policy-abc",
			Name:        "round-trip-test",
			Description: "Testing round-trip conversion",
			TeamId:      "team-123",
			Enabled:     true,
			ClusterIds:  []string{"c1", "c2", "c3", "c4"},
		}

		// Convert to model
		model := &NodePolicyTargetResourceModel{}
		model.fromProto(original)

		// Convert back to proto
		ctx := context.Background()
		diags := diag.Diagnostics{}
		converted := model.toProto(ctx, &diags, "team-123")

		if diags.HasError() {
			t.Fatalf("Round-trip conversion had errors: %v", diags)
		}

		// Verify all fields match
		if converted.TargetId != original.TargetId {
			t.Errorf("TargetId mismatch: expected %s, got %s", original.TargetId, converted.TargetId)
		}
		if converted.PolicyId != original.PolicyId {
			t.Errorf("PolicyId mismatch: expected %s, got %s", original.PolicyId, converted.PolicyId)
		}
		if converted.Name != original.Name {
			t.Errorf("Name mismatch: expected %s, got %s", original.Name, converted.Name)
		}
		if converted.Description != original.Description {
			t.Errorf("Description mismatch: expected %s, got %s", original.Description, converted.Description)
		}
		if converted.Enabled != original.Enabled {
			t.Errorf("Enabled mismatch: expected %v, got %v", original.Enabled, converted.Enabled)
		}
		if len(converted.ClusterIds) != len(original.ClusterIds) {
			t.Errorf("ClusterIds length mismatch: expected %d, got %d", len(original.ClusterIds), len(converted.ClusterIds))
		}
		for i, id := range original.ClusterIds {
			if i < len(converted.ClusterIds) && converted.ClusterIds[i] != id {
				t.Errorf("ClusterIds[%d] mismatch: expected %s, got %s", i, id, converted.ClusterIds[i])
			}
		}
	})
}

func validateNodePolicyTargetSchema(t *testing.T, schema schema.Schema) {
	// Validate required attributes
	requiredAttrs := []string{"policy_id", "name", "cluster_ids"}
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

	// Validate optional attributes
	optionalAttrs := []string{"description", "enabled"}
	for _, attr := range optionalAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Optional attribute %s not found in schema", attr)
		}
	}

	// Validate that required attributes are actually required
	if attr, exists := schema.Attributes["policy_id"]; exists {
		if !attr.IsRequired() {
			t.Error("policy_id should be required")
		}
	}

	if attr, exists := schema.Attributes["name"]; exists {
		if !attr.IsRequired() {
			t.Error("name should be required")
		}
	}

	if attr, exists := schema.Attributes["cluster_ids"]; exists {
		if !attr.IsRequired() {
			t.Error("cluster_ids should be required")
		}
	}
}
