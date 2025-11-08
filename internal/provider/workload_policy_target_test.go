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

	// Test LabelSelector with multiple MatchExpressions and multiple values
	t.Run("LabelSelector_WithMultipleMatchExpressions", func(t *testing.T) {
		selector := &LabelSelector{
			MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"app":         types.StringValue("web"),
				"environment": types.StringValue("production"),
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
							"values": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("frontend"),
								types.StringValue("backend"),
								types.StringValue("middleware"),
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
							"key":      types.StringValue("region"),
							"operator": types.StringValue("NotIn"),
							"values": types.ListValueMust(types.StringType, []attr.Value{
								types.StringValue("us-east-1"),
								types.StringValue("us-west-1"),
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
							"key":      types.StringValue("version"),
							"operator": types.StringValue("Exists"),
							"values":   types.ListValueMust(types.StringType, []attr.Value{}),
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
		if proto.MatchLabels["app"] != "web" {
			t.Errorf("Expected app=web, got %s", proto.MatchLabels["app"])
		}
		if proto.MatchLabels["environment"] != "production" {
			t.Errorf("Expected environment=production, got %s", proto.MatchLabels["environment"])
		}
		if len(proto.MatchExpressions) != 3 {
			t.Fatalf("Expected 3 match expressions, got %d", len(proto.MatchExpressions))
		}

		// Check first expression (In with multiple values)
		expr1 := proto.MatchExpressions[0]
		if expr1.Key != "tier" {
			t.Errorf("Expected key=tier, got %s", expr1.Key)
		}
		if expr1.Operator != 1 { // IN = 1
			t.Errorf("Expected operator=IN (1), got %d", expr1.Operator)
		}
		if len(expr1.Values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(expr1.Values))
		}
		expectedTiers := []string{"frontend", "backend", "middleware"}
		for i, expected := range expectedTiers {
			if expr1.Values[i] != expected {
				t.Errorf("Expected value[%d]='%s', got '%s'", i, expected, expr1.Values[i])
			}
		}

		// Check second expression (NotIn with multiple values)
		expr2 := proto.MatchExpressions[1]
		if expr2.Key != "region" {
			t.Errorf("Expected key=region, got %s", expr2.Key)
		}
		if expr2.Operator != 2 { // NOT_IN = 2
			t.Errorf("Expected operator=NOT_IN (2), got %d", expr2.Operator)
		}
		if len(expr2.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(expr2.Values))
		}
		expectedRegions := []string{"us-east-1", "us-west-1"}
		for i, expected := range expectedRegions {
			if expr2.Values[i] != expected {
				t.Errorf("Expected value[%d]='%s', got '%s'", i, expected, expr2.Values[i])
			}
		}

		// Check third expression (Exists with no values)
		expr3 := proto.MatchExpressions[2]
		if expr3.Key != "version" {
			t.Errorf("Expected key=version, got %s", expr3.Key)
		}
		if expr3.Operator != 3 { // EXISTS = 3
			t.Errorf("Expected operator=EXISTS (3), got %d", expr3.Operator)
		}
		if len(expr3.Values) != 0 {
			t.Errorf("Expected 0 values for Exists operator, got %d", len(expr3.Values))
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

	// Test LabelSelector round-trip conversion (toProto -> fromProto)
	t.Run("LabelSelector_RoundTrip", func(t *testing.T) {
		original := &LabelSelector{
			MatchLabels: types.MapValueMust(types.StringType, map[string]attr.Value{
				"app": types.StringValue("web"),
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
							"key":      types.StringValue("tier"),
							"operator": types.StringValue("In"),
							"values":   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("frontend"), types.StringValue("backend")}),
						},
					),
					types.ObjectValueMust(
						map[string]attr.Type{
							"key":      types.StringType,
							"operator": types.StringType,
							"values":   types.ListType{ElemType: types.StringType},
						},
						map[string]attr.Value{
							"key":      types.StringValue("region"),
							"operator": types.StringValue("NotIn"),
							"values":   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("us-east-1")}),
						},
					),
					types.ObjectValueMust(
						map[string]attr.Type{
							"key":      types.StringType,
							"operator": types.StringType,
							"values":   types.ListType{ElemType: types.StringType},
						},
						map[string]attr.Value{
							"key":      types.StringValue("version"),
							"operator": types.StringValue("Exists"),
							"values":   types.ListValueMust(types.StringType, []attr.Value{}),
						},
					),
				},
			),
		}

		// Convert to proto
		ctx := context.Background()
		proto, err := original.toProto(ctx)
		if err != nil {
			t.Fatalf("toProto failed: %v", err)
		}

		// Convert back from proto
		converted := &LabelSelector{}
		converted.fromProto(proto)

		// Verify match labels
		if !original.MatchLabels.Equal(converted.MatchLabels) {
			t.Errorf("Match labels don't match after round-trip")
		}

		// Verify match expressions count
		if len(original.MatchExpressions.Elements()) != len(converted.MatchExpressions.Elements()) {
			t.Fatalf("Expected %d match expressions, got %d", len(original.MatchExpressions.Elements()), len(converted.MatchExpressions.Elements()))
		}

		// Verify each match expression
		for i, origElem := range original.MatchExpressions.Elements() {
			origObj := origElem.(types.Object)
			convObj := converted.MatchExpressions.Elements()[i].(types.Object)

			if !origObj.Equal(convObj) {
				t.Errorf("Match expression %d doesn't match after round-trip", i)
			}
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
