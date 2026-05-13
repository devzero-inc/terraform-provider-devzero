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

func TestWorkloadRuleResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	NewWorkloadRuleResource().Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema had errors: %v", resp.Diagnostics)
	}

	validateWorkloadRuleSchema(t, resp.Schema)
}

func validateWorkloadRuleSchema(t *testing.T, s schema.Schema) {
	t.Helper()

	requiredAttrs := []string{"cluster_id", "namespace", "kind", "name"}
	for _, attr := range requiredAttrs {
		a, exists := s.Attributes[attr]
		if !exists {
			t.Errorf("Required attribute %q not found in schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Attribute %q should be required", attr)
		}
	}

	computedAttrs := []string{"id"}
	for _, attr := range computedAttrs {
		a, exists := s.Attributes[attr]
		if !exists {
			t.Errorf("Computed attribute %q not found in schema", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Attribute %q should be computed", attr)
		}
	}

	optionalAttrs := []string{
		"auto_generate",
		"cpu_rule", "memory_rule", "gpu_rule",
		"hpa_rule", "emergency_response",
		"action_triggers", "startup_period_seconds", "cron_schedule", "cooldown_minutes",
		"detection_triggers", "scheduler_plugins", "defragmentation_schedule",
		"live_migration_enabled", "use_in_place_vertical_scaling",
		"containers",
	}
	for _, attr := range optionalAttrs {
		if _, exists := s.Attributes[attr]; !exists {
			t.Errorf("Optional attribute %q not found in schema", attr)
		}
	}
}

func TestWorkloadRuleResourceModel(t *testing.T) {
	t.Parallel()

	// ---------- ResourceRuleConfigModel ----------

	t.Run("ResourceRuleConfigModel_ToProto_AllFields", func(t *testing.T) {
		m := &ResourceRuleConfigModel{
			Enabled:                 types.BoolValue(true),
			MinRequest:              types.Int64Value(100),
			MaxRequest:              types.Int64Value(4000),
			LimitMultiplier:         types.Float32Value(1.5),
			LimitsAdjustmentEnabled: types.BoolValue(true),
			TargetPercentile:        types.Float32Value(0.95),
			MaxScaleUpPercent:       types.Float32Value(50.0),
			MaxScaleDownPercent:     types.Float32Value(20.0),
			LimitsRemovalEnabled:    types.BoolValue(false),
		}

		p := m.toProto()
		if p == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !p.Enabled {
			t.Error("Expected Enabled=true")
		}
		if p.MinRequest == nil || *p.MinRequest != 100 {
			t.Errorf("Expected MinRequest=100, got %v", p.MinRequest)
		}
		if p.MaxRequest == nil || *p.MaxRequest != 4000 {
			t.Errorf("Expected MaxRequest=4000, got %v", p.MaxRequest)
		}
		if p.LimitMultiplier == nil || *p.LimitMultiplier != 1.5 {
			t.Errorf("Expected LimitMultiplier=1.5, got %v", p.LimitMultiplier)
		}
		if !p.LimitsAdjustmentEnabled {
			t.Error("Expected LimitsAdjustmentEnabled=true")
		}
		if p.TargetPercentile == nil || *p.TargetPercentile != 0.95 {
			t.Errorf("Expected TargetPercentile=0.95, got %v", p.TargetPercentile)
		}
		if p.MaxScaleUpPercent == nil || *p.MaxScaleUpPercent != 50.0 {
			t.Errorf("Expected MaxScaleUpPercent=50.0, got %v", p.MaxScaleUpPercent)
		}
		if p.MaxScaleDownPercent == nil || *p.MaxScaleDownPercent != 20.0 {
			t.Errorf("Expected MaxScaleDownPercent=20.0, got %v", p.MaxScaleDownPercent)
		}
		if p.LimitsRemovalEnabled {
			t.Error("Expected LimitsRemovalEnabled=false")
		}
	})

	t.Run("ResourceRuleConfigModel_ToProto_NilWhenNil", func(t *testing.T) {
		var m *ResourceRuleConfigModel
		if m.toProto() != nil {
			t.Error("Expected nil proto from nil model")
		}
	})

	t.Run("ResourceRuleConfigModel_ToProto_NullOptionals", func(t *testing.T) {
		m := &ResourceRuleConfigModel{
			Enabled:                 types.BoolValue(false),
			MinRequest:              types.Int64Null(),
			MaxRequest:              types.Int64Null(),
			LimitMultiplier:         types.Float32Null(),
			LimitsAdjustmentEnabled: types.BoolValue(false),
			TargetPercentile:        types.Float32Null(),
			MaxScaleUpPercent:       types.Float32Null(),
			MaxScaleDownPercent:     types.Float32Null(),
			LimitsRemovalEnabled:    types.BoolValue(false),
		}
		p := m.toProto()
		if p == nil {
			t.Fatal("Expected non-nil proto")
		}
		if p.MinRequest != nil {
			t.Errorf("Expected nil MinRequest, got %v", p.MinRequest)
		}
		if p.MaxRequest != nil {
			t.Errorf("Expected nil MaxRequest, got %v", p.MaxRequest)
		}
		if p.LimitMultiplier != nil {
			t.Errorf("Expected nil LimitMultiplier, got %v", p.LimitMultiplier)
		}
		if p.TargetPercentile != nil {
			t.Errorf("Expected nil TargetPercentile, got %v", p.TargetPercentile)
		}
		if p.MaxScaleUpPercent != nil {
			t.Errorf("Expected nil MaxScaleUpPercent, got %v", p.MaxScaleUpPercent)
		}
		if p.MaxScaleDownPercent != nil {
			t.Errorf("Expected nil MaxScaleDownPercent, got %v", p.MaxScaleDownPercent)
		}
	})

	t.Run("ResourceRuleConfigFromProto_AllFields", func(t *testing.T) {
		minReq := int64(100)
		maxReq := int64(4000)
		limitMul := float32(1.5)
		targetPct := float32(0.95)
		scaleUp := float32(50.0)
		scaleDown := float32(20.0)

		p := &apiv1.ResourceRuleConfig{
			Enabled:                 true,
			MinRequest:              &minReq,
			MaxRequest:              &maxReq,
			LimitMultiplier:         &limitMul,
			LimitsAdjustmentEnabled: true,
			TargetPercentile:        &targetPct,
			MaxScaleUpPercent:       &scaleUp,
			MaxScaleDownPercent:     &scaleDown,
			LimitsRemovalEnabled:    false,
		}

		m := resourceRuleConfigFromProto(p)
		if m == nil {
			t.Fatal("Expected non-nil model")
		}
		if !m.Enabled.ValueBool() {
			t.Error("Expected Enabled=true")
		}
		if m.MinRequest.ValueInt64() != 100 {
			t.Errorf("Expected MinRequest=100, got %d", m.MinRequest.ValueInt64())
		}
		if m.MaxRequest.ValueInt64() != 4000 {
			t.Errorf("Expected MaxRequest=4000, got %d", m.MaxRequest.ValueInt64())
		}
		if m.LimitMultiplier.ValueFloat32() != 1.5 {
			t.Errorf("Expected LimitMultiplier=1.5, got %f", m.LimitMultiplier.ValueFloat32())
		}
		if !m.LimitsAdjustmentEnabled.ValueBool() {
			t.Error("Expected LimitsAdjustmentEnabled=true")
		}
		if m.TargetPercentile.ValueFloat32() != 0.95 {
			t.Errorf("Expected TargetPercentile=0.95, got %f", m.TargetPercentile.ValueFloat32())
		}
		if m.MaxScaleUpPercent.ValueFloat32() != 50.0 {
			t.Errorf("Expected MaxScaleUpPercent=50.0, got %f", m.MaxScaleUpPercent.ValueFloat32())
		}
		if m.MaxScaleDownPercent.ValueFloat32() != 20.0 {
			t.Errorf("Expected MaxScaleDownPercent=20.0, got %f", m.MaxScaleDownPercent.ValueFloat32())
		}
	})

	t.Run("ResourceRuleConfigFromProto_Nil", func(t *testing.T) {
		if resourceRuleConfigFromProto(nil) != nil {
			t.Error("Expected nil model from nil proto")
		}
	})

	t.Run("ResourceRuleConfigFromProto_NilOptionals", func(t *testing.T) {
		p := &apiv1.ResourceRuleConfig{Enabled: false}
		m := resourceRuleConfigFromProto(p)
		if m == nil {
			t.Fatal("Expected non-nil model")
		}
		if !m.MinRequest.IsNull() {
			t.Error("Expected MinRequest to be null")
		}
		if !m.MaxRequest.IsNull() {
			t.Error("Expected MaxRequest to be null")
		}
		if !m.LimitMultiplier.IsNull() {
			t.Error("Expected LimitMultiplier to be null")
		}
		if !m.TargetPercentile.IsNull() {
			t.Error("Expected TargetPercentile to be null")
		}
		if !m.MaxScaleUpPercent.IsNull() {
			t.Error("Expected MaxScaleUpPercent to be null")
		}
		if !m.MaxScaleDownPercent.IsNull() {
			t.Error("Expected MaxScaleDownPercent to be null")
		}
	})

	// ---------- HPARuleConfigModel ----------

	t.Run("HPARuleConfigModel_ToProto_AllFields", func(t *testing.T) {
		m := &HPARuleConfigModel{
			Enabled:                 types.BoolValue(true),
			MinReplicas:             types.Int32Value(2),
			MaxReplicas:             types.Int32Value(10),
			TargetUtilization:       types.Float32Value(0.7),
			PrimaryMetric:           types.StringValue("cpu"),
			MaxReplicaChangePercent: types.Float32Value(50.0),
		}

		p := m.toProto()
		if p == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !p.Enabled {
			t.Error("Expected Enabled=true")
		}
		if p.MinReplicas == nil || *p.MinReplicas != 2 {
			t.Errorf("Expected MinReplicas=2, got %v", p.MinReplicas)
		}
		if p.MaxReplicas == nil || *p.MaxReplicas != 10 {
			t.Errorf("Expected MaxReplicas=10, got %v", p.MaxReplicas)
		}
		if p.TargetUtilization == nil || *p.TargetUtilization != 0.7 {
			t.Errorf("Expected TargetUtilization=0.7, got %v", p.TargetUtilization)
		}
		if p.PrimaryMetric == nil || *p.PrimaryMetric != apiv1.HPAMetricType_HPA_METRIC_TYPE_CPU {
			t.Errorf("Expected PrimaryMetric=CPU, got %v", p.PrimaryMetric)
		}
		if p.MaxReplicaChangePercent == nil || *p.MaxReplicaChangePercent != 50.0 {
			t.Errorf("Expected MaxReplicaChangePercent=50.0, got %v", p.MaxReplicaChangePercent)
		}
	})

	t.Run("HPARuleConfigModel_ToProto_NilWhenNil", func(t *testing.T) {
		var m *HPARuleConfigModel
		if m.toProto() != nil {
			t.Error("Expected nil proto from nil model")
		}
	})

	t.Run("HPARuleConfigFromProto_AllFields", func(t *testing.T) {
		minR := int32(2)
		maxR := int32(10)
		util := float32(0.7)
		metric := apiv1.HPAMetricType_HPA_METRIC_TYPE_MEMORY
		maxChange := float32(50.0)

		p := &apiv1.HPARuleConfig{
			Enabled:                 true,
			MinReplicas:             &minR,
			MaxReplicas:             &maxR,
			TargetUtilization:       &util,
			PrimaryMetric:           &metric,
			MaxReplicaChangePercent: &maxChange,
		}

		m := hpaRuleConfigFromProto(p)
		if m == nil {
			t.Fatal("Expected non-nil model")
		}
		if !m.Enabled.ValueBool() {
			t.Error("Expected Enabled=true")
		}
		if m.MinReplicas.ValueInt32() != 2 {
			t.Errorf("Expected MinReplicas=2, got %d", m.MinReplicas.ValueInt32())
		}
		if m.MaxReplicas.ValueInt32() != 10 {
			t.Errorf("Expected MaxReplicas=10, got %d", m.MaxReplicas.ValueInt32())
		}
		if m.TargetUtilization.ValueFloat32() != 0.7 {
			t.Errorf("Expected TargetUtilization=0.7, got %f", m.TargetUtilization.ValueFloat32())
		}
		if m.PrimaryMetric.ValueString() != "memory" {
			t.Errorf("Expected PrimaryMetric='memory', got %s", m.PrimaryMetric.ValueString())
		}
		if m.MaxReplicaChangePercent.ValueFloat32() != 50.0 {
			t.Errorf("Expected MaxReplicaChangePercent=50.0, got %f", m.MaxReplicaChangePercent.ValueFloat32())
		}
	})

	t.Run("HPARuleConfigFromProto_Nil", func(t *testing.T) {
		if hpaRuleConfigFromProto(nil) != nil {
			t.Error("Expected nil model from nil proto")
		}
	})

	// ---------- HPA metric conversions ----------

	t.Run("HPAMetricToProto", func(t *testing.T) {
		cases := []struct {
			input    string
			expected apiv1.HPAMetricType
		}{
			{"cpu", apiv1.HPAMetricType_HPA_METRIC_TYPE_CPU},
			{"memory", apiv1.HPAMetricType_HPA_METRIC_TYPE_MEMORY},
			{"gpu", apiv1.HPAMetricType_HPA_METRIC_TYPE_GPU},
			{"network_ingress", apiv1.HPAMetricType_HPA_METRIC_TYPE_NETWORK_INGRESS},
			{"network_egress", apiv1.HPAMetricType_HPA_METRIC_TYPE_NETWORK_EGRESS},
		}
		for _, tc := range cases {
			result := wrHPAMetricToProto(tc.input)
			if result == nil {
				t.Errorf("Expected non-nil for %q", tc.input)
				continue
			}
			if *result != tc.expected {
				t.Errorf("Input %q: expected %v, got %v", tc.input, tc.expected, *result)
			}
		}
		if wrHPAMetricToProto("unknown") != nil {
			t.Error("Expected nil for unknown metric")
		}
	})

	t.Run("HPAMetricFromProto", func(t *testing.T) {
		cases := []struct {
			input    apiv1.HPAMetricType
			expected string
		}{
			{apiv1.HPAMetricType_HPA_METRIC_TYPE_CPU, "cpu"},
			{apiv1.HPAMetricType_HPA_METRIC_TYPE_MEMORY, "memory"},
			{apiv1.HPAMetricType_HPA_METRIC_TYPE_GPU, "gpu"},
			{apiv1.HPAMetricType_HPA_METRIC_TYPE_NETWORK_INGRESS, "network_ingress"},
			{apiv1.HPAMetricType_HPA_METRIC_TYPE_NETWORK_EGRESS, "network_egress"},
		}
		for _, tc := range cases {
			result := wrHPAMetricFromProto(tc.input)
			if result != tc.expected {
				t.Errorf("Input %v: expected %q, got %q", tc.input, tc.expected, result)
			}
		}
		if wrHPAMetricFromProto(apiv1.HPAMetricType_HPA_METRIC_TYPE_UNSPECIFIED) != "" {
			t.Error("Expected empty string for unspecified metric")
		}
	})

	// ---------- EmergencyResponseModel ----------

	t.Run("EmergencyResponseModel_ToProto", func(t *testing.T) {
		m := &EmergencyResponseModel{
			OomEnabled:              types.BoolValue(true),
			OomMemoryMultiplier:     types.Float32Value(2.0),
			OomMaxReactions:         types.Int32Value(3),
			OomCooldownSeconds:      types.Int32Value(60),
			CpuThrottlingEnabled:    types.BoolValue(true),
			CpuThrottlingThreshold:  types.Float32Value(0.8),
			CpuThrottlingMultiplier: types.Float32Value(1.5),
		}

		p := m.toProto()
		if p == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !p.OomEnabled {
			t.Error("Expected OomEnabled=true")
		}
		if p.OomMemoryMultiplier != 2.0 {
			t.Errorf("Expected OomMemoryMultiplier=2.0, got %f", p.OomMemoryMultiplier)
		}
		if p.OomMaxReactions != 3 {
			t.Errorf("Expected OomMaxReactions=3, got %d", p.OomMaxReactions)
		}
		if p.OomCooldownSeconds != 60 {
			t.Errorf("Expected OomCooldownSeconds=60, got %d", p.OomCooldownSeconds)
		}
		if !p.CpuThrottlingEnabled {
			t.Error("Expected CpuThrottlingEnabled=true")
		}
		if p.CpuThrottlingThreshold != 0.8 {
			t.Errorf("Expected CpuThrottlingThreshold=0.8, got %f", p.CpuThrottlingThreshold)
		}
		if p.CpuThrottlingMultiplier != 1.5 {
			t.Errorf("Expected CpuThrottlingMultiplier=1.5, got %f", p.CpuThrottlingMultiplier)
		}
	})

	t.Run("EmergencyResponseModel_ToProto_NilWhenNil", func(t *testing.T) {
		var m *EmergencyResponseModel
		if m.toProto() != nil {
			t.Error("Expected nil proto from nil model")
		}
	})

	t.Run("EmergencyResponseFromProto", func(t *testing.T) {
		p := &apiv1.EmergencyResponseConfig{
			OomEnabled:              true,
			OomMemoryMultiplier:     2.0,
			OomMaxReactions:         3,
			OomCooldownSeconds:      60,
			CpuThrottlingEnabled:    true,
			CpuThrottlingThreshold:  0.8,
			CpuThrottlingMultiplier: 1.5,
		}

		m := emergencyResponseFromProto(p)
		if m == nil {
			t.Fatal("Expected non-nil model")
		}
		if !m.OomEnabled.ValueBool() {
			t.Error("Expected OomEnabled=true")
		}
		if m.OomMemoryMultiplier.ValueFloat32() != 2.0 {
			t.Errorf("Expected OomMemoryMultiplier=2.0, got %f", m.OomMemoryMultiplier.ValueFloat32())
		}
		if m.OomMaxReactions.ValueInt32() != 3 {
			t.Errorf("Expected OomMaxReactions=3, got %d", m.OomMaxReactions.ValueInt32())
		}
		if m.OomCooldownSeconds.ValueInt32() != 60 {
			t.Errorf("Expected OomCooldownSeconds=60, got %d", m.OomCooldownSeconds.ValueInt32())
		}
		if !m.CpuThrottlingEnabled.ValueBool() {
			t.Error("Expected CpuThrottlingEnabled=true")
		}
		if m.CpuThrottlingThreshold.ValueFloat32() != 0.8 {
			t.Errorf("Expected CpuThrottlingThreshold=0.8, got %f", m.CpuThrottlingThreshold.ValueFloat32())
		}
		if m.CpuThrottlingMultiplier.ValueFloat32() != 1.5 {
			t.Errorf("Expected CpuThrottlingMultiplier=1.5, got %f", m.CpuThrottlingMultiplier.ValueFloat32())
		}
	})

	t.Run("EmergencyResponseFromProto_Nil", func(t *testing.T) {
		if emergencyResponseFromProto(nil) != nil {
			t.Error("Expected nil model from nil proto")
		}
	})

	// ---------- ContainerResourceConfigModel ----------

	t.Run("ContainerResourceConfigModel_ToProto", func(t *testing.T) {
		m := &ContainerResourceConfigModel{
			Enabled:                 types.BoolValue(true),
			MinRequest:              types.Int64Value(50),
			MaxRequest:              types.Int64Value(2000),
			LimitMultiplier:         types.Float32Value(2.0),
			LimitsAdjustmentEnabled: types.BoolValue(true),
			TargetPercentile:        types.Float32Value(0.9),
			LimitsRemovalEnabled:    types.BoolValue(false),
		}

		p := m.toProto()
		if p == nil {
			t.Fatal("Expected non-nil proto")
		}
		if !p.Enabled {
			t.Error("Expected Enabled=true")
		}
		if p.MinRequest == nil || *p.MinRequest != 50 {
			t.Errorf("Expected MinRequest=50, got %v", p.MinRequest)
		}
		if p.MaxRequest == nil || *p.MaxRequest != 2000 {
			t.Errorf("Expected MaxRequest=2000, got %v", p.MaxRequest)
		}
		if p.LimitMultiplier == nil || *p.LimitMultiplier != 2.0 {
			t.Errorf("Expected LimitMultiplier=2.0, got %v", p.LimitMultiplier)
		}
		if !p.LimitsAdjustmentEnabled {
			t.Error("Expected LimitsAdjustmentEnabled=true")
		}
		if p.TargetPercentile == nil || *p.TargetPercentile != 0.9 {
			t.Errorf("Expected TargetPercentile=0.9, got %v", p.TargetPercentile)
		}
		if p.LimitsRemovalEnabled {
			t.Error("Expected LimitsRemovalEnabled=false")
		}
	})

	t.Run("ContainerResourceConfigModel_ToProto_NilWhenNil", func(t *testing.T) {
		var m *ContainerResourceConfigModel
		if m.toProto() != nil {
			t.Error("Expected nil proto from nil model")
		}
	})

	t.Run("ContainerResourceConfigFromProto", func(t *testing.T) {
		minReq := int64(50)
		maxReq := int64(2000)
		limitMul := float32(2.0)
		targetPct := float32(0.9)

		p := &apiv1.ContainerResourceConfig{
			Enabled:                 true,
			MinRequest:              &minReq,
			MaxRequest:              &maxReq,
			LimitMultiplier:         &limitMul,
			LimitsAdjustmentEnabled: true,
			TargetPercentile:        &targetPct,
			LimitsRemovalEnabled:    false,
		}

		m := containerResourceConfigFromProto(p)
		if m == nil {
			t.Fatal("Expected non-nil model")
		}
		if !m.Enabled.ValueBool() {
			t.Error("Expected Enabled=true")
		}
		if m.MinRequest.ValueInt64() != 50 {
			t.Errorf("Expected MinRequest=50, got %d", m.MinRequest.ValueInt64())
		}
		if m.MaxRequest.ValueInt64() != 2000 {
			t.Errorf("Expected MaxRequest=2000, got %d", m.MaxRequest.ValueInt64())
		}
		if m.LimitMultiplier.ValueFloat32() != 2.0 {
			t.Errorf("Expected LimitMultiplier=2.0, got %f", m.LimitMultiplier.ValueFloat32())
		}
		if !m.LimitsAdjustmentEnabled.ValueBool() {
			t.Error("Expected LimitsAdjustmentEnabled=true")
		}
		if m.TargetPercentile.ValueFloat32() != 0.9 {
			t.Errorf("Expected TargetPercentile=0.9, got %f", m.TargetPercentile.ValueFloat32())
		}
	})

	t.Run("ContainerResourceConfigFromProto_Nil", func(t *testing.T) {
		if containerResourceConfigFromProto(nil) != nil {
			t.Error("Expected nil model from nil proto")
		}
	})

	t.Run("ContainerResourceConfigFromProto_NullOptionals", func(t *testing.T) {
		p := &apiv1.ContainerResourceConfig{Enabled: false}
		m := containerResourceConfigFromProto(p)
		if m == nil {
			t.Fatal("Expected non-nil model")
		}
		if !m.MinRequest.IsNull() {
			t.Error("Expected MinRequest to be null")
		}
		if !m.MaxRequest.IsNull() {
			t.Error("Expected MaxRequest to be null")
		}
		if !m.LimitMultiplier.IsNull() {
			t.Error("Expected LimitMultiplier to be null")
		}
		if !m.TargetPercentile.IsNull() {
			t.Error("Expected TargetPercentile to be null")
		}
	})

	// ---------- Container rule lists ----------

	t.Run("ContainerRuleModels_ToProto", func(t *testing.T) {
		containers := []ContainerRuleModel{
			{
				ContainerName: types.StringValue("main"),
				CpuRule: &ContainerResourceConfigModel{
					Enabled:                 types.BoolValue(true),
					MinRequest:              types.Int64Value(100),
					MaxRequest:              types.Int64Null(),
					LimitMultiplier:         types.Float32Null(),
					LimitsAdjustmentEnabled: types.BoolValue(false),
					TargetPercentile:        types.Float32Null(),
					LimitsRemovalEnabled:    types.BoolValue(false),
				},
				MemoryRule: nil,
				GpuRule:    nil,
			},
		}

		protos := containerRuleModelsToProto(containers)
		if len(protos) != 1 {
			t.Fatalf("Expected 1 proto, got %d", len(protos))
		}
		if protos[0].ContainerName != "main" {
			t.Errorf("Expected ContainerName='main', got %s", protos[0].ContainerName)
		}
		if protos[0].CpuRule == nil {
			t.Error("Expected non-nil CpuRule")
		}
		if protos[0].MemoryRule != nil {
			t.Error("Expected nil MemoryRule")
		}
		if protos[0].GpuRule != nil {
			t.Error("Expected nil GpuRule")
		}
	})

	t.Run("ContainerRuleModels_ToProto_Empty", func(t *testing.T) {
		if containerRuleModelsToProto(nil) != nil {
			t.Error("Expected nil from empty slice")
		}
		if containerRuleModelsToProto([]ContainerRuleModel{}) != nil {
			t.Error("Expected nil from empty slice")
		}
	})

	t.Run("ContainerRuleModels_FromProto", func(t *testing.T) {
		minReq := int64(100)
		protos := []*apiv1.ContainerResourceRuleConfig{
			{
				ContainerName: "sidecar",
				CpuRule:       &apiv1.ContainerResourceConfig{Enabled: true, MinRequest: &minReq},
				MemoryRule:    nil,
				GpuRule:       nil,
			},
		}

		models := containerRuleModelsFromProto(protos)
		if len(models) != 1 {
			t.Fatalf("Expected 1 model, got %d", len(models))
		}
		if models[0].ContainerName.ValueString() != "sidecar" {
			t.Errorf("Expected ContainerName='sidecar', got %s", models[0].ContainerName.ValueString())
		}
		if models[0].CpuRule == nil {
			t.Error("Expected non-nil CpuRule")
		}
		if models[0].MemoryRule != nil {
			t.Error("Expected nil MemoryRule")
		}
	})

	t.Run("ContainerRuleModels_FromProto_Empty", func(t *testing.T) {
		if containerRuleModelsFromProto(nil) != nil {
			t.Error("Expected nil from nil input")
		}
		if containerRuleModelsFromProto([]*apiv1.ContainerResourceRuleConfig{}) != nil {
			t.Error("Expected nil from empty slice")
		}
	})

	// ---------- WorkloadRuleResourceModel.toProto ----------

	t.Run("WorkloadRuleResourceModel_ToProto_AutoGenerate", func(t *testing.T) {
		ctx := context.Background()
		var diags diag.Diagnostics

		m := &WorkloadRuleResourceModel{
			ClusterId:                 types.StringValue("cluster-abc"),
			Namespace:                 types.StringValue("production"),
			Kind:                      types.StringValue("Deployment"),
			Name:                      types.StringValue("my-api"),
			AutoGenerate:              types.BoolValue(true),
			ActionTriggers:            types.ListValueMust(types.StringType, []attr.Value{}),
			DetectionTriggers:         types.ListValueMust(types.StringType, []attr.Value{}),
			SchedulerPlugins:          types.ListValueMust(types.StringType, []attr.Value{}),
			StartupPeriodSeconds:      types.Int64Null(),
			CronSchedule:              types.StringNull(),
			CooldownMinutes:           types.Int32Null(),
			DefragmentationSchedule:   types.StringNull(),
			LiveMigrationEnabled:      types.BoolValue(false),
			UseInPlaceVerticalScaling: types.BoolValue(false),
		}

		req := m.toProto(ctx, &diags, "team-123")
		if diags.HasError() {
			t.Fatalf("Unexpected error: %v", diags)
		}
		if req == nil {
			t.Fatal("Expected non-nil request")
		}
		if req.TeamId != "team-123" {
			t.Errorf("Expected TeamId='team-123', got %s", req.TeamId)
		}
		if req.ClusterId != "cluster-abc" {
			t.Errorf("Expected ClusterId='cluster-abc', got %s", req.ClusterId)
		}
		if req.Namespace != "production" {
			t.Errorf("Expected Namespace='production', got %s", req.Namespace)
		}
		if req.Kind != "Deployment" {
			t.Errorf("Expected Kind='Deployment', got %s", req.Kind)
		}
		if req.Name != "my-api" {
			t.Errorf("Expected Name='my-api', got %s", req.Name)
		}
		if !req.AutoGenerate {
			t.Error("Expected AutoGenerate=true")
		}
		// When auto_generate=true, Fields should be nil
		if req.Fields != nil {
			t.Error("Expected Fields=nil when auto_generate=true")
		}
	})

	t.Run("WorkloadRuleResourceModel_ToProto_ManualFields", func(t *testing.T) {
		ctx := context.Background()
		var diags diag.Diagnostics

		cronSchedule := "0 2 * * *"
		startupPeriod := int64(300)
		cooldown := int32(60)

		m := &WorkloadRuleResourceModel{
			ClusterId:    types.StringValue("cluster-xyz"),
			Namespace:    types.StringValue("default"),
			Kind:         types.StringValue("StatefulSet"),
			Name:         types.StringValue("my-db"),
			AutoGenerate: types.BoolValue(false),
			CpuRule: &ResourceRuleConfigModel{
				Enabled:                 types.BoolValue(true),
				MinRequest:              types.Int64Value(200),
				MaxRequest:              types.Int64Null(),
				LimitMultiplier:         types.Float32Null(),
				LimitsAdjustmentEnabled: types.BoolValue(false),
				TargetPercentile:        types.Float32Null(),
				MaxScaleUpPercent:       types.Float32Null(),
				MaxScaleDownPercent:     types.Float32Null(),
				LimitsRemovalEnabled:    types.BoolValue(false),
			},
			MemoryRule:        nil,
			GpuRule:           nil,
			HpaRule:           nil,
			EmergencyResponse: nil,
			ActionTriggers: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("on_schedule"),
			}),
			StartupPeriodSeconds: types.Int64Value(startupPeriod),
			CronSchedule:         types.StringValue(cronSchedule),
			CooldownMinutes:      types.Int32Value(cooldown),
			DetectionTriggers: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("pod_creation"),
			}),
			SchedulerPlugins:          types.ListValueMust(types.StringType, []attr.Value{}),
			DefragmentationSchedule:   types.StringNull(),
			LiveMigrationEnabled:      types.BoolValue(false),
			UseInPlaceVerticalScaling: types.BoolValue(true),
			Containers:                nil,
		}

		req := m.toProto(ctx, &diags, "team-456")
		if diags.HasError() {
			t.Fatalf("Unexpected error: %v", diags)
		}
		if req == nil {
			t.Fatal("Expected non-nil request")
		}
		if req.AutoGenerate {
			t.Error("Expected AutoGenerate=false")
		}
		if req.Fields == nil {
			t.Fatal("Expected non-nil Fields when auto_generate=false")
		}
		if len(req.Fields.ActionTriggers) != 1 || req.Fields.ActionTriggers[0] != apiv1.ActionTrigger_ACTION_TRIGGER_ON_SCHEDULE {
			t.Errorf("Expected ActionTriggers=[ON_SCHEDULE], got %v", req.Fields.ActionTriggers)
		}
		if req.Fields.StartupPeriodSeconds == nil || *req.Fields.StartupPeriodSeconds != 300 {
			t.Errorf("Expected StartupPeriodSeconds=300, got %v", req.Fields.StartupPeriodSeconds)
		}
		if req.Fields.CronSchedule == nil || *req.Fields.CronSchedule != "0 2 * * *" {
			t.Errorf("Expected CronSchedule='0 2 * * *', got %v", req.Fields.CronSchedule)
		}
		if req.Fields.CooldownMinutes == nil || *req.Fields.CooldownMinutes != 60 {
			t.Errorf("Expected CooldownMinutes=60, got %v", req.Fields.CooldownMinutes)
		}
		if len(req.Fields.DetectionTriggers) != 1 || req.Fields.DetectionTriggers[0] != apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION {
			t.Errorf("Expected DetectionTriggers=[POD_CREATION], got %v", req.Fields.DetectionTriggers)
		}
		if req.Fields.CpuRule == nil {
			t.Error("Expected non-nil CpuRule in Fields")
		}
		if req.Fields.UseInPlaceVerticalScaling != true {
			t.Error("Expected UseInPlaceVerticalScaling=true")
		}
		if req.Fields.DefragmentationSchedule != nil {
			t.Errorf("Expected nil DefragmentationSchedule, got %v", req.Fields.DefragmentationSchedule)
		}
	})

	// ---------- WorkloadRuleResourceModel.fromProto ----------

	t.Run("WorkloadRuleResourceModel_FromProto_ManualSource", func(t *testing.T) {
		cronSchedule := "0 2 * * *"
		startupPeriod := int64(300)
		cooldown := int32(60)
		defragSchedule := "0 3 * * 0"

		r := &apiv1.WorkloadRule{
			RuleId:        "rule-001",
			ClusterId:     "cluster-abc",
			Namespace:     "production",
			Kind:          "Deployment",
			Name:          "my-api",
			CurrentSource: "manual",
			ActionTriggers: []apiv1.ActionTrigger{
				apiv1.ActionTrigger_ACTION_TRIGGER_ON_DETECTION,
			},
			StartupPeriodSeconds: &startupPeriod,
			CronSchedule:         &cronSchedule,
			CooldownMinutes:      &cooldown,
			DetectionTriggers: []apiv1.WorkloadDetectionTrigger{
				apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION,
				apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_UPDATE,
			},
			SchedulerPlugins:          []string{"binpacking"},
			DefragmentationSchedule:   &defragSchedule,
			LiveMigrationEnabled:      true,
			UseInPlaceVerticalScaling: false,
		}

		var m WorkloadRuleResourceModel
		m.fromProto(r)

		if m.Id.ValueString() != "rule-001" {
			t.Errorf("Expected Id='rule-001', got %s", m.Id.ValueString())
		}
		if m.ClusterId.ValueString() != "cluster-abc" {
			t.Errorf("Expected ClusterId='cluster-abc', got %s", m.ClusterId.ValueString())
		}
		if m.AutoGenerate.ValueBool() {
			t.Error("Expected AutoGenerate=false for manual source")
		}
		if m.StartupPeriodSeconds.ValueInt64() != 300 {
			t.Errorf("Expected StartupPeriodSeconds=300, got %d", m.StartupPeriodSeconds.ValueInt64())
		}
		if m.CronSchedule.ValueString() != "0 2 * * *" {
			t.Errorf("Expected CronSchedule='0 2 * * *', got %s", m.CronSchedule.ValueString())
		}
		if m.CooldownMinutes.ValueInt32() != 60 {
			t.Errorf("Expected CooldownMinutes=60, got %d", m.CooldownMinutes.ValueInt32())
		}
		if m.DefragmentationSchedule.ValueString() != "0 3 * * 0" {
			t.Errorf("Expected DefragmentationSchedule='0 3 * * 0', got %s", m.DefragmentationSchedule.ValueString())
		}
		if !m.LiveMigrationEnabled.ValueBool() {
			t.Error("Expected LiveMigrationEnabled=true")
		}
		if m.UseInPlaceVerticalScaling.ValueBool() {
			t.Error("Expected UseInPlaceVerticalScaling=false")
		}

		// Verify action triggers
		if m.ActionTriggers.IsNull() || len(m.ActionTriggers.Elements()) != 1 {
			t.Fatalf("Expected 1 action trigger, got %v", m.ActionTriggers)
		}

		// Verify detection triggers
		if m.DetectionTriggers.IsNull() || len(m.DetectionTriggers.Elements()) != 2 {
			t.Fatalf("Expected 2 detection triggers, got %v", m.DetectionTriggers)
		}

		// Verify scheduler plugins
		if m.SchedulerPlugins.IsNull() || len(m.SchedulerPlugins.Elements()) != 1 {
			t.Fatalf("Expected 1 scheduler plugin, got %v", m.SchedulerPlugins)
		}
	})

	t.Run("WorkloadRuleResourceModel_FromProto_AutoOptimization", func(t *testing.T) {
		r := &apiv1.WorkloadRule{
			RuleId:        "rule-auto",
			ClusterId:     "cluster-abc",
			Namespace:     "default",
			Kind:          "Deployment",
			Name:          "auto-app",
			CurrentSource: "auto_optimization",
		}

		var m WorkloadRuleResourceModel
		m.fromProto(r)

		if !m.AutoGenerate.ValueBool() {
			t.Error("Expected AutoGenerate=true for auto_optimization source")
		}
	})

	t.Run("WorkloadRuleResourceModel_FromProto_NilOptionals", func(t *testing.T) {
		r := &apiv1.WorkloadRule{
			RuleId:        "rule-002",
			ClusterId:     "cluster-abc",
			Namespace:     "default",
			Kind:          "Deployment",
			Name:          "my-app",
			CurrentSource: "manual",
		}

		var m WorkloadRuleResourceModel
		m.fromProto(r)

		if !m.StartupPeriodSeconds.IsNull() {
			t.Error("Expected StartupPeriodSeconds to be null")
		}
		if !m.CronSchedule.IsNull() {
			t.Error("Expected CronSchedule to be null")
		}
		if !m.CooldownMinutes.IsNull() {
			t.Error("Expected CooldownMinutes to be null")
		}
		if !m.DefragmentationSchedule.IsNull() {
			t.Error("Expected DefragmentationSchedule to be null")
		}
		if m.CpuRule != nil {
			t.Error("Expected CpuRule to be nil")
		}
		if m.HpaRule != nil {
			t.Error("Expected HpaRule to be nil")
		}
		if m.EmergencyResponse != nil {
			t.Error("Expected EmergencyResponse to be nil")
		}
		if m.Containers != nil {
			t.Error("Expected Containers to be nil")
		}
	})

	t.Run("WorkloadRuleResourceModel_FromProto_WithContainers", func(t *testing.T) {
		minReq := int64(100)
		r := &apiv1.WorkloadRule{
			RuleId:        "rule-003",
			ClusterId:     "cluster-abc",
			Namespace:     "default",
			Kind:          "Deployment",
			Name:          "my-app",
			CurrentSource: "manual",
			Containers: []*apiv1.ContainerResourceRuleConfig{
				{
					ContainerName: "app",
					CpuRule:       &apiv1.ContainerResourceConfig{Enabled: true, MinRequest: &minReq},
				},
				{
					ContainerName: "sidecar",
				},
			},
		}

		var m WorkloadRuleResourceModel
		m.fromProto(r)

		if len(m.Containers) != 2 {
			t.Fatalf("Expected 2 containers, got %d", len(m.Containers))
		}
		if m.Containers[0].ContainerName.ValueString() != "app" {
			t.Errorf("Expected container[0].Name='app', got %s", m.Containers[0].ContainerName.ValueString())
		}
		if m.Containers[0].CpuRule == nil {
			t.Error("Expected non-nil CpuRule for container[0]")
		}
		if m.Containers[1].ContainerName.ValueString() != "sidecar" {
			t.Errorf("Expected container[1].Name='sidecar', got %s", m.Containers[1].ContainerName.ValueString())
		}
	})
}
