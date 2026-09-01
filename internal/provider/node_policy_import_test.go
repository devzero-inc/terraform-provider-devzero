package provider

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// TestAccNodePolicyResource_ImportRoundTrip proves full import fidelity for
// devzero_node_policy: the policy is seeded directly against the fake
// backend (never via a Terraform apply), so importing it is the only way
// Terraform ever learns about it — exactly the "resource created by any
// means, Terraform must still be able to import it cleanly" scenario. It
// deliberately sets an azure.image_version-only nested block (the exact
// shape that used to be dropped by the isAzureSpecEmpty bug) to prove that
// fix holds through a real import.
func TestAccNodePolicyResource_ImportRoundTrip(t *testing.T) {
	factories, fake := testAccHarness(t)

	imageVersion := "AzureLinux-202401.01.0"
	seeded, err := fake.CreateNodePolicies(context.Background(), connect.NewRequest(&apiv1.CreateNodePoliciesRequest{
		TeamId: testAccTeamID,
		Policies: []*apiv1.NodePolicy{
			{
				Name:        "acc-node-policy",
				Description: "seeded outside terraform",
				Azure: &apiv1.AzureNodeClassSpec{
					ImageVersion: &imageVersion,
				},
			},
		},
	}))
	if err != nil {
		t.Fatalf("seeding node policy: %s", err)
	}
	policyID := seeded.Msg.Policies[0].Id

	importCfg := testAccProviderConfig() + `
resource "devzero_node_policy" "test" {
  name        = "acc-node-policy"
  description = "seeded outside terraform"
  azure = {
    image_version = "AzureLinux-202401.01.0"
  }
}
`
	renamedCfg := testAccProviderConfig() + `
resource "devzero_node_policy" "test" {
  name        = "acc-node-policy-renamed"
  description = "seeded outside terraform"
  azure = {
    image_version = "AzureLinux-202401.01.0"
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:             importCfg,
				ResourceName:       "devzero_node_policy.test",
				ImportState:        true,
				ImportStateId:      policyID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					attrs := states[0].Attributes
					if got := attrs["name"]; got != "acc-node-policy" {
						return fmt.Errorf("imported name = %q, want %q", got, "acc-node-policy")
					}
					if got := attrs["azure.image_version"]; got != imageVersion {
						return fmt.Errorf("imported azure.image_version = %q, want %q", got, imageVersion)
					}
					return nil
				},
			},
			{
				// Same config as import, applied fresh right after: must
				// produce zero changes.
				Config:   importCfg,
				PlanOnly: true,
			},
			{
				Config: renamedCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("devzero_node_policy.test", "name", "acc-node-policy-renamed"),
					testAccCheckNodePolicyNameInBackend(fake, "devzero_node_policy.test", "acc-node-policy-renamed"),
				),
			},
		},
	})
}

func testAccCheckNodePolicyNameInBackend(fake *fakeBackend, resourceName, wantName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		id := rs.Primary.ID

		fake.mu.Lock()
		defer fake.mu.Unlock()
		p, ok := fake.nodePol[id]
		if !ok {
			return fmt.Errorf("node policy %s not found in fake backend", id)
		}
		if p.Name != wantName {
			return fmt.Errorf("fake backend node policy %s has name %q, want %q", id, p.Name, wantName)
		}
		return nil
	}
}
