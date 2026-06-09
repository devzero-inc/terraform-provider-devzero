package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkloadPolicyResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Instantiate the resource and call Schema
	workloadPolicyResource := NewWorkloadPolicyResource()
	workloadPolicyResource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	// Validate the schema
	validateSchema(t, resp.Schema)
}

func TestWorkloadPolicyResourceModel(t *testing.T) {
	t.Parallel()

	// Test VerticalScalingOptions
	t.Run("VerticalScalingOptions", func(t *testing.T) {
		opts := &VerticalScalingOptions{
			Enabled:                 types.BoolValue(true),
			MinRequest:              types.Int64Value(100),
			MaxRequest:              types.Int64Value(1000),
			OverheadMultiplier:      types.Float32Value(0.1),
			LimitsAdjustmentEnabled: types.BoolValue(true),
			TargetPercentile:        types.Float32Value(0.75),
			MaxScaleUpPercent:       types.Float32Value(50.0),
			MaxScaleDownPercent:     types.Float32Value(25.0),
			LimitMultiplier:         types.Float32Value(2.0),
			MinDataPoints:           types.Int32Value(10),
			AdjustReqEvenIfNotSet:   types.BoolValue(true),
			LimitsRemovalEnabled:    types.BoolValue(true),
		}

		// Test toProto
		proto := opts.toProto()
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !proto.Enabled {
			t.Error("Expected Enabled to be true")
		}
		if *proto.MinRequest != 100 {
			t.Errorf("Expected MinRequest to be 100, got %d", *proto.MinRequest)
		}
		if *proto.MaxRequest != 1000 {
			t.Errorf("Expected MaxRequest to be 1000, got %d", *proto.MaxRequest)
		}
		if *proto.OverheadMultiplier != 0.1 {
			t.Errorf("Expected OverheadMultiplier to be 0.1, got %f", *proto.OverheadMultiplier)
		}
		if *proto.LimitsAdjustmentEnabled != true {
			t.Error("Expected LimitsAdjustmentEnabled to be true")
		}
		if *proto.TargetPercentile != 0.75 {
			t.Errorf("Expected TargetPercentile to be 0.75, got %f", *proto.TargetPercentile)
		}
		if *proto.MaxScaleUpPercent != 50.0 {
			t.Errorf("Expected MaxScaleUpPercent to be 50.0, got %f", *proto.MaxScaleUpPercent)
		}
		if *proto.MaxScaleDownPercent != 25.0 {
			t.Errorf("Expected MaxScaleDownPercent to be 25.0, got %f", *proto.MaxScaleDownPercent)
		}
		if *proto.LimitMultiplier != 2.0 {
			t.Errorf("Expected LimitMultiplier to be 2.0, got %f", *proto.LimitMultiplier)
		}
		if *proto.MinDataPoints != 10 {
			t.Errorf("Expected MinDataPoints to be 10, got %d", *proto.MinDataPoints)
		}
		if !proto.AdjustReqEvenIfNotSet {
			t.Error("Expected AdjustReqEvenIfNotSet to be true")
		}
		if !proto.LimitsRemovalEnabled {
			t.Error("Expected LimitsRemovalEnabled to be true")
		}
	})

	// Test VerticalScalingOptions new fields default to false
	t.Run("VerticalScalingOptionsDefaults", func(t *testing.T) {
		opts := &VerticalScalingOptions{
			Enabled: types.BoolValue(true),
		}
		proto := opts.toProto()
		if proto.AdjustReqEvenIfNotSet {
			t.Error("Expected AdjustReqEvenIfNotSet to default to false")
		}
		if proto.LimitsRemovalEnabled {
			t.Error("Expected LimitsRemovalEnabled to default to false")
		}
	})

	// Test HorizontalScalingOptions
	t.Run("HorizontalScalingOptions", func(t *testing.T) {
		opts := &HorizontalScalingOptions{
			Enabled:                 types.BoolValue(true),
			MinReplicas:             types.Int32Value(1),
			MaxReplicas:             types.Int32Value(10),
			TargetUtilization:       types.Float32Value(0.7),
			PrimaryMetric:           types.StringValue("cpu"),
			MinDataPoints:           types.Int32Value(5),
			MaxReplicaChangePercent: types.Float32Value(100.0),
		}

		// Test toProto
		proto := opts.toProto()
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !proto.Enabled {
			t.Error("Expected Enabled to be true")
		}
		if *proto.MinReplicas != 1 {
			t.Errorf("Expected MinReplicas to be 1, got %d", *proto.MinReplicas)
		}
		if *proto.MaxReplicas != 10 {
			t.Errorf("Expected MaxReplicas to be 10, got %d", *proto.MaxReplicas)
		}
		if *proto.TargetUtilization != 0.7 {
			t.Errorf("Expected TargetUtilization to be 0.7, got %f", *proto.TargetUtilization)
		}
		if *proto.PrimaryMetric != 1 { // CPU = 1
			t.Errorf("Expected PrimaryMetric to be CPU (1), got %d", *proto.PrimaryMetric)
		}
		if *proto.MinDataPoints != 5 {
			t.Errorf("Expected MinDataPoints to be 5, got %d", *proto.MinDataPoints)
		}
		if *proto.MaxReplicaChangePercent != 100.0 {
			t.Errorf("Expected MaxReplicaChangePercent to be 100.0, got %f", *proto.MaxReplicaChangePercent)
		}
	})

	// Test pmax protection fields on WorkloadPolicyResourceModel
	t.Run("PmaxProtection", func(t *testing.T) {
		ctx := context.Background()
		var diagnostics diag.Diagnostics

		model := &WorkloadPolicyResourceModel{
			Name:                    types.StringValue("test"),
			Description:             types.StringValue(""),
			CronSchedule:            types.StringValue("*/15 * * * *"),
			DefragmentationSchedule: types.StringValue("*/15 * * * *"),
			ActionTriggers:          types.ListValueMust(types.StringType, nil),
			DetectionTriggers:       types.ListValueMust(types.StringType, nil),
			SchedulerPlugins:        types.ListValueMust(types.StringType, nil),
			EnablePmaxProtection:    types.BoolValue(true),
			PmaxRatioThreshold:      types.Float32Value(3.0),
		}

		proto := model.toProto(ctx, &diagnostics, "team-123")
		if proto == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !proto.EnablePmaxProtection {
			t.Error("Expected EnablePmaxProtection to be true")
		}
		if proto.PmaxRatioThreshold == nil || *proto.PmaxRatioThreshold != 3.0 {
			t.Errorf("Expected PmaxRatioThreshold to be 3.0, got %v", proto.PmaxRatioThreshold)
		}
	})

	// Test HPAMetric conversions
	t.Run("HPAMetricConversions", func(t *testing.T) {
		opts := &HorizontalScalingOptions{}

		// Test toHPAMetric
		opts.PrimaryMetric = types.StringValue("cpu")
		metric := opts.toHPAMetric()
		if metric == nil {
			t.Fatal("Expected non-nil metric")
		}
		if *metric != 1 { // CPU = 1
			t.Errorf("Expected CPU metric (1), got %d", *metric)
		}

		// Test fromHPAMetric
		result := opts.fromHPAMetric(1) // CPU
		if result != "cpu" {
			t.Errorf("Expected 'cpu', got %s", result)
		}

		result = opts.fromHPAMetric(2) // Memory
		if result != "memory" {
			t.Errorf("Expected 'memory', got %s", result)
		}

		result = opts.fromHPAMetric(3) // GPU
		if result != "gpu" {
			t.Errorf("Expected 'gpu', got %s", result)
		}

		result = opts.fromHPAMetric(4) // Network
		if result != "network" {
			t.Errorf("Expected 'network', got %s", result)
		}
	})
}

func validateSchema(t *testing.T, s schema.Schema) {
	// Validate required attributes
	requiredAttrs := []string{"name", "action_triggers"}
	for _, attr := range requiredAttrs {
		if _, exists := s.Attributes[attr]; !exists {
			t.Errorf("Required attribute %s not found in schema", attr)
		}
	}

	// Validate computed attributes
	computedAttrs := []string{"id"}
	for _, attr := range computedAttrs {
		if attrSchema, exists := s.Attributes[attr]; exists {
			if !attrSchema.IsComputed() {
				t.Errorf("Attribute %s should be computed", attr)
			}
		}
	}

	// Validate nested attributes exist
	if _, exists := s.Attributes["cpu_vertical_scaling"]; !exists {
		t.Error("cpu_vertical_scaling attribute not found")
	}

	if _, exists := s.Attributes["horizontal_scaling"]; !exists {
		t.Error("horizontal_scaling attribute not found")
	}

	// Validate pmax fields exist at top level
	for _, attr := range []string{"enable_pmax_protection", "pmax_ratio_threshold"} {
		if _, exists := s.Attributes[attr]; !exists {
			t.Errorf("Expected top-level attribute %q not found in schema", attr)
		}
	}

	// Validate new vertical scaling fields exist inside cpu_vertical_scaling
	if cpuVs, ok := s.Attributes["cpu_vertical_scaling"].(schema.SingleNestedAttribute); ok {
		for _, attr := range []string{"adjust_req_even_if_not_set", "limits_removal_enabled"} {
			if _, exists := cpuVs.Attributes[attr]; !exists {
				t.Errorf("Expected vertical_scaling attribute %q not found in cpu_vertical_scaling", attr)
			}
		}
	}
}
