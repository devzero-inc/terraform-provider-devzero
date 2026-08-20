package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// leafOverrides supplies valid values for attributes whose converters only
// accept a fixed vocabulary (enums). Keyed by attribute name, applied at any
// nesting depth. Anything not listed gets a generic "x" / 1 / true.
var leafOverrides = map[string]tftypes.Value{
	"action_triggers":    tftypes.NewValue(tftypes.String, "on_schedule"),
	"detection_triggers": tftypes.NewValue(tftypes.String, "pod_creation"),
	"operator":           tftypes.NewValue(tftypes.String, "In"),
	"effect":             tftypes.NewValue(tftypes.String, "NoSchedule"),
	"kinds":              tftypes.NewValue(tftypes.String, "Deployment"),
	"workload_kinds":     tftypes.NewValue(tftypes.String, "Deployment"),
}

// populatedValue builds a tftypes.Value of type typ in which every object,
// list, set and map is NON-null (lists/sets get one element, maps one entry)
// and every primitive leaf holds a concrete value ("x", 1, true) unless
// leafOverrides names the attribute. Decoding such a value into a resource
// model exercises every tfsdk struct at every nesting level, which is exactly
// what catches "Struct defines fields not found in object" / "Object defines
// fields not found in struct" mismatches between a schema and its Go model —
// the bug class that broke `azure {}` / `aws {}` in v0.1.6 — and then drives
// every toProto/fromProto branch.
func populatedValue(typ tftypes.Type) tftypes.Value {
	return populatedValueNamed("", typ)
}

func populatedValueNamed(name string, typ tftypes.Type) tftypes.Value {
	switch t := typ.(type) {
	case tftypes.Object:
		vals := make(map[string]tftypes.Value, len(t.AttributeTypes))
		for attrName, at := range t.AttributeTypes {
			vals[attrName] = populatedValueNamed(attrName, at)
		}
		return tftypes.NewValue(t, vals)
	case tftypes.List:
		return tftypes.NewValue(t, []tftypes.Value{populatedValueNamed(name, t.ElementType)})
	case tftypes.Set:
		return tftypes.NewValue(t, []tftypes.Value{populatedValueNamed(name, t.ElementType)})
	case tftypes.Map:
		return tftypes.NewValue(t, map[string]tftypes.Value{"k": populatedValueNamed(name, t.ElementType)})
	case tftypes.Tuple:
		vals := make([]tftypes.Value, len(t.ElementTypes))
		for i, et := range t.ElementTypes {
			vals[i] = populatedValueNamed(name, et)
		}
		return tftypes.NewValue(t, vals)
	}
	if v, ok := leafOverrides[name]; ok && v.Type().Equal(typ) {
		return v
	}
	switch {
	case typ.Is(tftypes.String):
		return tftypes.NewValue(typ, "x")
	case typ.Is(tftypes.Number):
		return tftypes.NewValue(typ, 1)
	case typ.Is(tftypes.Bool):
		return tftypes.NewValue(typ, true)
	}
	return tftypes.NewValue(typ, nil)
}

func resourceSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

type resourceUnderTest struct {
	name     string
	resource func() resource.Resource
	newModel func() any
	// roundTrip converts the decoded model to its proto and back (nil = skip).
	// It must return the model populated by fromProto.
	roundTrip func(ctx context.Context, model any) (any, diag.Diagnostics)
}

func allResourcesUnderTest() []resourceUnderTest {
	return []resourceUnderTest{
		{
			name:     "devzero_node_policy",
			resource: NewNodePolicyResource,
			newModel: func() any { return &NodePolicyResourceModel{} },
			roundTrip: func(ctx context.Context, model any) (any, diag.Diagnostics) {
				var d diag.Diagnostics
				m := model.(*NodePolicyResourceModel)
				p := m.toProto(ctx, &d, "team")
				if d.HasError() {
					return nil, d
				}
				p.Id = "id"
				var back NodePolicyResourceModel
				back.fromProto(p)
				return &back, d
			},
		},
		{
			name:     "devzero_node_policy_target",
			resource: NewNodePolicyTargetResource,
			newModel: func() any { return &NodePolicyTargetResourceModel{} },
			roundTrip: func(ctx context.Context, model any) (any, diag.Diagnostics) {
				var d diag.Diagnostics
				m := model.(*NodePolicyTargetResourceModel)
				p := m.toProto(ctx, &d, "team")
				if d.HasError() {
					return nil, d
				}
				var back NodePolicyTargetResourceModel
				back.fromProto(p)
				return &back, d
			},
		},
		{
			name:     "devzero_workload_policy",
			resource: NewWorkloadPolicyResource,
			newModel: func() any { return &WorkloadPolicyResourceModel{} },
			roundTrip: func(ctx context.Context, model any) (any, diag.Diagnostics) {
				var d diag.Diagnostics
				m := model.(*WorkloadPolicyResourceModel)
				p := m.toProto(ctx, &d, "team")
				if d.HasError() {
					return nil, d
				}
				var back WorkloadPolicyResourceModel
				back.fromProto(p)
				return &back, d
			},
		},
		{
			name:     "devzero_workload_policy_target",
			resource: NewWorkloadPolicyTargetResource,
			newModel: func() any { return &WorkloadPolicyTargetResourceModel{} },
		},
		{
			name:     "devzero_workload_rule",
			resource: NewWorkloadRuleResource,
			newModel: func() any { return &WorkloadRuleResourceModel{} },
		},
		{
			name:     "devzero_cluster",
			resource: NewClusterResource,
			newModel: func() any { return &ClusterResourceModel{} },
		},
	}
}

// TestSchemaModelConsistency decodes a fully-populated value built from each
// resource's real schema into its Go model (what Terraform does on
// Create/Update), then round-trips it through toProto/fromProto and sets it
// back into state (what happens after every API call).
func TestSchemaModelConsistency(t *testing.T) {
	ctx := context.Background()
	for _, rt := range allResourcesUnderTest() {
		t.Run(rt.name, func(t *testing.T) {
			s := resourceSchema(t, rt.resource())
			objType := s.Type().TerraformType(ctx)

			plan := tfsdk.Plan{Schema: s, Raw: populatedValue(objType)}
			model := rt.newModel()
			if d := plan.Get(ctx, model); d.HasError() {
				t.Fatalf("Plan.Get into %T failed (schema/model mismatch): %v", model, d)
			}

			// Everything decoded must be settable back into state unchanged.
			state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}
			if d := state.Set(ctx, model); d.HasError() {
				t.Fatalf("State.Set from decoded %T failed: %v", model, d)
			}

			if rt.roundTrip == nil {
				return
			}
			back, d := rt.roundTrip(ctx, model)
			if d.HasError() {
				t.Fatalf("toProto/fromProto round-trip failed: %v", d)
			}
			state = tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}
			if d := state.Set(ctx, back); d.HasError() {
				t.Fatalf("State.Set after fromProto failed (fromProto produced a value the schema cannot hold): %v", d)
			}
		})
	}
}

// TestSchemaModelConsistency_AllNull is the complementary case: a config where
// every optional attribute is omitted must decode and round-trip cleanly too.
func TestSchemaModelConsistency_AllNull(t *testing.T) {
	ctx := context.Background()
	for _, rt := range allResourcesUnderTest() {
		t.Run(rt.name, func(t *testing.T) {
			s := resourceSchema(t, rt.resource())
			objType := s.Type().TerraformType(ctx).(tftypes.Object)
			vals := map[string]tftypes.Value{}
			for name, at := range objType.AttributeTypes {
				vals[name] = tftypes.NewValue(at, nil)
			}
			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objType, vals)}
			model := rt.newModel()
			if d := plan.Get(ctx, model); d.HasError() {
				t.Fatalf("Plan.Get into %T failed: %v", model, d)
			}
			if rt.roundTrip == nil {
				return
			}
			back, d := rt.roundTrip(ctx, model)
			if d.HasError() {
				t.Fatalf("toProto/fromProto round-trip failed: %v", d)
			}
			state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}
			if d := state.Set(ctx, back); d.HasError() {
				t.Fatalf("State.Set after fromProto failed: %v", d)
			}
		})
	}
}
