package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkloadPolicyTargetResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Instantiate the resource and call Schema
	workloadPolicyTargetResource := NewWorkloadPolicyTargetResource()
	workloadPolicyTargetResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	// Validate the schema
	validateTargetSchema(t, resp.Schema)
}

func TestWorkloadPolicyTargetResourceModel(t *testing.T) {
	t.Parallel()

	// Test LabelSelector
	t.Run("LabelSelector", func(t *testing.T) {
		selector := &LabelSelector{
			MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"app": types.StringValue("api"),
				"env": types.StringValue("prod"),
			}),
			MatchExpressions: types.ListValueMust(types.StringType, []attr.Value{}), // Simplified for testing
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
							"key":      types.StringValue("tier"),
							"operator": types.StringValue("In"),
							"values":   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("frontend"), types.StringValue("backend")}),
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
		if expr.Key != "tier" {
			t.Errorf("Expected key=tier, got %s", expr.Key)
		}
		if expr.Operator != 1 { // IN = 1
			t.Errorf("Expected operator=IN (1), got %d", expr.Operator)
		}
		if len(expr.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(expr.Values))
		}
	})

	// Test RegexPattern
	t.Run("RegexPattern", func(t *testing.T) {
		pattern := &RegexPattern{
			Pattern: types.StringValue("^api-.*$"),
			Flags:   types.StringValue("i"),
		}

		// Test toProto
		proto := pattern.toProto()
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if proto.Pattern != "^api-.*$" {
			t.Errorf("Expected pattern '^api-.*$', got %s", proto.Pattern)
		}
		if proto.Flags != "i" {
			t.Errorf("Expected flags 'i', got %s", proto.Flags)
		}
	})

	// Test MatchExpression
	t.Run("MatchExpression", func(t *testing.T) {
		expr := &MatchExpression{
			Key:      types.StringValue("tier"),
			Operator: types.StringValue("In"),
			Values:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("frontend"), types.StringValue("backend")}),
		}

		// Test toProto
		ctx := context.Background()
		proto, err := expr.toProto(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if proto.Key != "tier" {
			t.Errorf("Expected key=tier, got %s", proto.Key)
		}
		if proto.Operator != 1 { // IN = 1
			t.Errorf("Expected operator=IN (1), got %d", proto.Operator)
		}
		if len(proto.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(proto.Values))
		}
	})
}

func validateTargetSchema(t *testing.T, schema schema.Schema) {
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

	// Validate nested attributes exist
	if _, exists := schema.Attributes["namespace_selector"]; !exists {
		t.Error("namespace_selector attribute not found")
	}

	if _, exists := schema.Attributes["workload_selector"]; !exists {
		t.Error("workload_selector attribute not found")
	}

	if _, exists := schema.Attributes["name_pattern"]; !exists {
		t.Error("name_pattern attribute not found")
	}
}
