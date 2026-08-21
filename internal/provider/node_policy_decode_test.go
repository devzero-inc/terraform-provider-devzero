package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// nodePolicySchemaPlan builds a tfsdk.Plan from the real resource schema, the same
// way Terraform core hands a planned value to Create/Update. Every top-level
// attribute not present in `set` is null. Nested objects are built with
// nullObjectWith so they carry every attribute the schema defines.
func nodePolicySchemaPlan(t *testing.T, set map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	r := &NodePolicyResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		if v, ok := set[name]; ok {
			vals[name] = v
			continue
		}
		vals[name] = tftypes.NewValue(typ, nil)
	}
	return tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

func nodePolicyAttrType(t *testing.T, path ...string) tftypes.Type {
	t.Helper()
	ctx := context.Background()
	r := &NodePolicyResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	typ := schemaResp.Schema.Type().TerraformType(ctx)
	for _, p := range path {
		switch tt := typ.(type) {
		case tftypes.Object:
			typ = tt.AttributeTypes[p]
		case tftypes.List:
			typ = tt.ElementType
		default:
			t.Fatalf("unexpected type %T at %q", typ, p)
		}
		if typ == nil {
			t.Fatalf("attribute %v not in schema", path)
		}
	}
	return typ
}

// nullObjectWith returns an object of type typ with all attributes null except those in set.
func nullObjectWith(t *testing.T, typ tftypes.Type, set map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	obj, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("type %v is not an object", typ)
	}
	vals := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, at := range obj.AttributeTypes {
		if v, ok := set[name]; ok {
			vals[name] = v
		} else {
			vals[name] = tftypes.NewValue(at, nil)
		}
	}
	return tftypes.NewValue(obj, vals)
}

// Regression for: "Struct defines fields not found in object: kubelet" when an
// `azure {}` block is set (v0.1.6). The Go model had a `kubelet` field the schema
// did not declare, so req.Plan.Get failed before any API call.
func TestNodePolicy_PlanDecode_AzureBlock(t *testing.T) {
	ctx := context.Background()
	azureType := nodePolicyAttrType(t, "azure")
	plan := nodePolicySchemaPlan(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "azure-pool"),
		"azure": nullObjectWith(t, azureType, map[string]tftypes.Value{
			"vnet_subnet_id": tftypes.NewValue(tftypes.String, "/subscriptions/x/subnet/y"),
		}),
	})

	var m NodePolicyResourceModel
	diags := plan.Get(ctx, &m)
	if diags.HasError() {
		t.Fatalf("plan.Get failed: %v", diags)
	}
	if m.Azure == nil {
		t.Fatalf("m.Azure is nil")
	}
	if got := m.Azure.VnetSubnetId.ValueString(); got != "/subscriptions/x/subnet/y" {
		t.Fatalf("got %q, want %s", got, "/subscriptions/x/subnet/y")
	}

	var d diag.Diagnostics
	proto := m.toProto(ctx, &d, "team-1")
	if d.HasError() {
		t.Fatalf("toProto failed: %v", d)
	}
	if proto.Azure == nil {
		t.Fatalf("proto.Azure is nil")
	}
	if got := proto.Azure.GetVnetSubnetId(); got != "/subscriptions/x/subnet/y" {
		t.Fatalf("got %q, want %s", got, "/subscriptions/x/subnet/y")
	}
}

// Regression for: "Unable to convert taints: can't unmarshal into *provider.Taint,
// needs FromTerraform5Value method" (v0.1.6). getElementList decoded list elements
// through tftypes.Value.As, which cannot populate tfsdk-tagged structs.
func TestNodePolicy_PlanDecode_Taints(t *testing.T) {
	ctx := context.Background()
	taintsType, ok := nodePolicyAttrType(t, "taints").(tftypes.List)
	if !ok {
		t.Fatal("taints attribute is not a list")
	}
	taint := nullObjectWith(t, taintsType.ElementType, map[string]tftypes.Value{
		"key":    tftypes.NewValue(tftypes.String, "dedicated"),
		"value":  tftypes.NewValue(tftypes.String, "gpu"),
		"effect": tftypes.NewValue(tftypes.String, "NoSchedule"),
	})
	plan := nodePolicySchemaPlan(t, map[string]tftypes.Value{
		"name":   tftypes.NewValue(tftypes.String, "tainted-pool"),
		"taints": tftypes.NewValue(taintsType, []tftypes.Value{taint}),
	})

	var m NodePolicyResourceModel
	diags := plan.Get(ctx, &m)
	if diags.HasError() {
		t.Fatalf("plan.Get failed: %v", diags)
	}

	var d diag.Diagnostics
	proto := m.toProto(ctx, &d, "team-1")
	if d.HasError() {
		t.Fatalf("toProto failed: %v", d)
	}
	if len(proto.Taints) != 1 {
		t.Fatalf("len(proto.Taints) = %d, want 1", len(proto.Taints))
	}
	if got := proto.Taints[0].Key; got != "dedicated" {
		t.Fatalf("got %q, want %s", got, "dedicated")
	}
	if got := proto.Taints[0].Value; got != "gpu" {
		t.Fatalf("got %q, want %s", got, "gpu")
	}
	if got := proto.Taints[0].Effect; got != "NoSchedule" {
		t.Fatalf("got %q, want %s", got, "NoSchedule")
	}
}

// The same struct/schema mismatch existed for `aws {}` (kubelet,
// capacity_reservation_selector_terms, context were in the model but not the schema).
func TestNodePolicy_PlanDecode_AwsBlock(t *testing.T) {
	ctx := context.Background()
	awsType := nodePolicyAttrType(t, "aws")
	plan := nodePolicySchemaPlan(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "aws-pool"),
		"aws": nullObjectWith(t, awsType, map[string]tftypes.Value{
			"ami_family": tftypes.NewValue(tftypes.String, "AL2023"),
		}),
	})

	var m NodePolicyResourceModel
	diags := plan.Get(ctx, &m)
	if diags.HasError() {
		t.Fatalf("plan.Get failed: %v", diags)
	}
	if m.Aws == nil {
		t.Fatalf("m.Aws is nil")
	}

	var d diag.Diagnostics
	proto := m.toProto(ctx, &d, "team-1")
	if d.HasError() {
		t.Fatalf("toProto failed: %v", d)
	}
	if proto.Aws == nil {
		t.Fatalf("proto.Aws is nil")
	}
	if got := proto.Aws.GetAmiFamily(); got != "AL2023" {
		t.Fatalf("got %q, want %s", got, "AL2023")
	}
}

// Raw karpenter specs also flow through getElementList with a struct element type.
func TestNodePolicy_PlanDecode_Raw(t *testing.T) {
	ctx := context.Background()
	rawType, ok := nodePolicyAttrType(t, "raw").(tftypes.List)
	if !ok {
		t.Fatal("raw attribute is not a list")
	}
	spec := nullObjectWith(t, rawType.ElementType, map[string]tftypes.Value{
		"nodepool_yaml":  tftypes.NewValue(tftypes.String, "apiVersion: karpenter.sh/v1\nkind: NodePool\n"),
		"nodeclass_yaml": tftypes.NewValue(tftypes.String, ""),
	})
	plan := nodePolicySchemaPlan(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "raw-pool"),
		"raw":  tftypes.NewValue(rawType, []tftypes.Value{spec}),
	})

	var m NodePolicyResourceModel
	diags := plan.Get(ctx, &m)
	if diags.HasError() {
		t.Fatalf("plan.Get failed: %v", diags)
	}

	var d diag.Diagnostics
	proto := m.toProto(ctx, &d, "team-1")
	if d.HasError() {
		t.Fatalf("toProto failed: %v", d)
	}
	if len(proto.Raw) != 1 {
		t.Fatalf("len(proto.Raw) = %d, want 1", len(proto.Raw))
	}
	if !strings.Contains(proto.Raw[0].NodepoolYaml, "kind: NodePool") {
		t.Fatalf("%q does not contain %s", proto.Raw[0].NodepoolYaml, "kind: NodePool")
	}
}

// Round-trip guard: whatever fromProto produces must be settable into state
// through the real schema (State.Set uses the same reflection as Plan.Get).
func TestNodePolicy_StateSet_RoundTrip(t *testing.T) {
	ctx := context.Background()
	azureType := nodePolicyAttrType(t, "azure")
	taintsType, ok := nodePolicyAttrType(t, "taints").(tftypes.List)
	if !ok {
		t.Fatal("taints attribute is not a list")
	}
	plan := nodePolicySchemaPlan(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "rt"),
		"azure": nullObjectWith(t, azureType, map[string]tftypes.Value{
			"vnet_subnet_id": tftypes.NewValue(tftypes.String, "subnet"),
		}),
		"taints": tftypes.NewValue(taintsType, []tftypes.Value{
			nullObjectWith(t, taintsType.ElementType, map[string]tftypes.Value{
				"key":    tftypes.NewValue(tftypes.String, "k"),
				"value":  tftypes.NewValue(tftypes.String, "v"),
				"effect": tftypes.NewValue(tftypes.String, "NoExecute"),
			}),
		}),
	})

	var m NodePolicyResourceModel
	if d0 := plan.Get(ctx, &m); d0.HasError() {
		t.Fatalf("plan.Get failed: %v", d0)
	}
	var d diag.Diagnostics
	proto := m.toProto(ctx, &d, "team-1")
	if d.HasError() {
		t.Fatalf("toProto failed: %v", d)
	}
	proto.Id = "np-123"

	var back NodePolicyResourceModel
	back.fromProto(proto)

	state := tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Schema.Type().TerraformType(ctx), nil)}
	setDiags := state.Set(ctx, &back)
	if setDiags.HasError() {
		t.Fatalf("state.Set failed: %v", setDiags)
	}
}
