package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// TestWorkloadRuleRead_ImportHydratesTriggerLists proves Read hydrates
// action_triggers/detection_triggers/scheduler_plugins from the real backend
// record right after import.
//
// Read's preserveNullsFrom step nulls these three fields whenever the PRIOR
// state had them null, to keep ordinary refreshes free of perpetual diffs
// for these plain-Optional (non-Computed) attributes. But right after
// `terraform import`, the prior state is import-shaped: only `id` is set and
// everything else is null — not because config said so, but because there
// is no config yet. preserveNullsFrom can't tell the difference, so it
// unconditionally wipes real backend data on import: a full-fidelity import
// bug.
func TestWorkloadRuleRead_ImportHydratesTriggerLists(t *testing.T) {
	ctx := context.Background()
	cs, fake := newFakeClientSet(t)
	r := &WorkloadRuleResource{client: cs}

	mutate := func(m *WorkloadRuleResourceModel) {
		m.ClusterId = types.StringValue("cluster-1")
		m.Namespace = types.StringValue("prod")
		m.Kind = types.StringValue("Deployment")
		m.Name = types.StringValue("api")
		m.Disabled = types.BoolValue(false)
		cpu := &ResourceRuleConfigModel{}
		hydrateNested(t, r, "cpu_rule", cpu)
		cpu.Enabled = types.BoolValue(true)
		cpu.MinRequest = types.Int64Value(10)
		m.CpuRule = cpu
	}
	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[WorkloadRuleResourceModel](t, r, mutate)}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)

	var created WorkloadRuleResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))
	id := created.Id.ValueString()

	// Give the backend record real trigger data, simulating a rule whose
	// triggers were set outside of this Terraform config (e.g. imported
	// from a rule the dashboard or another tool configured).
	fake.mu.Lock()
	fake.rules[id].ActionTriggers = []apiv1.ActionTrigger{
		apiv1.ActionTrigger_ACTION_TRIGGER_ON_SCHEDULE,
		apiv1.ActionTrigger_ACTION_TRIGGER_ON_DETECTION,
	}
	fake.rules[id].DetectionTriggers = []apiv1.WorkloadDetectionTrigger{
		apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION,
	}
	fake.rules[id].SchedulerPlugins = []string{"my-scheduler-plugin"}
	fake.mu.Unlock()

	// Build an import-shaped prior state: only `id` populated, matching
	// what resource.ImportStatePassthroughID produces before Read runs.
	importState := emptyState(t, r)
	mustNoDiags(t, "SetAttribute(id)", importState.SetAttribute(ctx, path.Root("id"), id))

	readResp := resource.ReadResponse{State: importState}
	r.Read(ctx, resource.ReadRequest{State: importState}, &readResp)
	mustNoDiags(t, "Read", readResp.Diagnostics)

	var got WorkloadRuleResourceModel
	mustNoDiags(t, "State.Get", readResp.State.Get(ctx, &got))

	if got.ActionTriggers.IsNull() {
		t.Fatal("action_triggers: expected hydrated list from import, got null")
	}
	if got.DetectionTriggers.IsNull() {
		t.Fatal("detection_triggers: expected hydrated list from import, got null")
	}
	if got.SchedulerPlugins.IsNull() {
		t.Fatal("scheduler_plugins: expected hydrated list from import, got null")
	}

	var gotActionTriggers []string
	mustNoDiags(t, "ActionTriggers.ElementsAs", got.ActionTriggers.ElementsAs(ctx, &gotActionTriggers, false))
	if len(gotActionTriggers) != 2 {
		t.Fatalf("action_triggers = %v, want 2 elements", gotActionTriggers)
	}

	var gotSchedulerPlugins []string
	mustNoDiags(t, "SchedulerPlugins.ElementsAs", got.SchedulerPlugins.ElementsAs(ctx, &gotSchedulerPlugins, false))
	if len(gotSchedulerPlugins) != 1 || gotSchedulerPlugins[0] != "my-scheduler-plugin" {
		t.Fatalf("scheduler_plugins = %v, want [my-scheduler-plugin]", gotSchedulerPlugins)
	}
}
