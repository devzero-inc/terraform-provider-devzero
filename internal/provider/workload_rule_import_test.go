package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// TestAccWorkloadRuleResource_ImportRoundTrip proves full import fidelity for
// devzero_workload_rule, including the trigger-list attributes
// (action_triggers/detection_triggers/scheduler_plugins). Those three are
// Optional+Computed with an empty-list Default, so they carry the exact
// systemic bug fixed by the preserve*StateOverDefault plan modifiers: with a
// non-empty real value and config omitting the attribute, a plan without the
// fix would revert them to `[]` on every apply. The rule is seeded directly
// against the fake backend (never via a Terraform apply), simulating a rule
// created by any means other than this Terraform config.
func TestAccWorkloadRuleResource_ImportRoundTrip(t *testing.T) {
	factories, fake := testAccHarness(t)

	ruleID := fake.id("wr")
	fake.mu.Lock()
	fake.rules[ruleID] = &apiv1.WorkloadRule{
		RuleId:    ruleID,
		ClusterId: "cluster-1",
		Namespace: "prod",
		Kind:      "Deployment",
		Name:      "api",
		CpuRule: &apiv1.ResourceRuleConfig{
			Enabled:    true,
			MinRequest: int64Ptr(10),
		},
		ActionTriggers: []apiv1.ActionTrigger{
			apiv1.ActionTrigger_ACTION_TRIGGER_ON_SCHEDULE,
			apiv1.ActionTrigger_ACTION_TRIGGER_ON_DETECTION,
		},
		DetectionTriggers: []apiv1.WorkloadDetectionTrigger{
			apiv1.WorkloadDetectionTrigger_DETECTION_TRIGGER_POD_CREATION,
		},
		SchedulerPlugins: []string{"my-scheduler-plugin"},
	}
	fake.mu.Unlock()

	importCfg := testAccProviderConfig() + `
resource "devzero_workload_rule" "test" {
  cluster_id = "cluster-1"
  namespace  = "prod"
  kind       = "Deployment"
  name       = "api"

  action_triggers    = ["on_schedule", "on_detection"]
  detection_triggers = ["pod_creation"]
  scheduler_plugins  = ["my-scheduler-plugin"]

  cpu_rule = {
    enabled     = true
    min_request = 10
  }
}
`
	updatedCfg := testAccProviderConfig() + `
resource "devzero_workload_rule" "test" {
  cluster_id = "cluster-1"
  namespace  = "prod"
  kind       = "Deployment"
  name       = "api"

  action_triggers    = ["on_schedule", "on_detection"]
  detection_triggers = ["pod_creation"]
  scheduler_plugins  = ["my-scheduler-plugin"]

  cpu_rule = {
    enabled     = true
    min_request = 25
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:             importCfg,
				ResourceName:       "devzero_workload_rule.test",
				ImportState:        true,
				ImportStateId:      ruleID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					attrs := states[0].Attributes
					if got := attrs["scheduler_plugins.0"]; got != "my-scheduler-plugin" {
						return fmt.Errorf("imported scheduler_plugins.0 = %q, want %q", got, "my-scheduler-plugin")
					}
					if got := attrs["action_triggers.#"]; got != "2" {
						return fmt.Errorf("imported action_triggers.# = %q, want 2", got)
					}
					return nil
				},
			},
			{
				// Same config as import, applied fresh right after: must
				// produce zero changes — proves the trigger lists (and
				// cpu_rule.min_request, which also carries a Default) don't
				// get clobbered back to their schema defaults.
				Config:   importCfg,
				PlanOnly: true,
			},
			{
				// Edit-and-reapply proof: mutate a genuinely mutable
				// attribute (min_request), not the workload identity
				// (cluster_id/namespace/kind/name) the backend upserts on.
				Config: updatedCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("devzero_workload_rule.test", "cpu_rule.min_request", "25"),
					testAccCheckWorkloadRuleMinRequestInBackend(fake, "devzero_workload_rule.test", 25),
				),
			},
		},
	})
}

func int64Ptr(v int64) *int64 { return &v }

func testAccCheckWorkloadRuleMinRequestInBackend(fake *fakeBackend, resourceName string, want int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		id := rs.Primary.ID

		fake.mu.Lock()
		defer fake.mu.Unlock()
		r, ok := fake.rules[id]
		if !ok {
			return fmt.Errorf("workload rule %s not found in fake backend", id)
		}
		if r.CpuRule == nil || r.CpuRule.MinRequest == nil {
			return fmt.Errorf("fake backend workload rule %s has no cpu_rule.min_request", id)
		}
		if got := *r.CpuRule.MinRequest; got != want {
			return fmt.Errorf("fake backend workload rule %s has cpu_rule.min_request %d, want %d", id, got, want)
		}
		return nil
	}
}
