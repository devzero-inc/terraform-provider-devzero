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

// TestAccClusterResource_ImportRoundTrip proves full import fidelity for
// devzero_cluster: import hydrates every readable attribute, the plan right
// after import is clean, and editing+reapplying an imported resource updates
// the real resource.
//
// The cluster is seeded directly against the fake backend (not via a
// Terraform apply) and imported as the test's first step. This is
// deliberate: a Terraform-driven Create never exercises Read at all (it
// writes state straight from the API response), so an import test built on
// top of an in-suite apply would still pass even if Read silently dropped
// fields. Importing a backend object Terraform never created is the only
// way to actually exercise "Read from scratch" the way `terraform import`
// really uses it, and it matches terraform-plugin-testing's own documented
// pattern for import-as-first-step (see ImportStatePersist's doc comment).
func TestAccClusterResource_ImportRoundTrip(t *testing.T) {
	factories, fake := testAccHarness(t)

	seeded, err := fake.CreateCluster(context.Background(), connect.NewRequest(&apiv1.CreateClusterRequest{
		TeamId:      testAccTeamID,
		ClusterName: "acc-cluster",
	}))
	if err != nil {
		t.Fatalf("seeding cluster: %s", err)
	}
	clusterID := seeded.Msg.Cluster.Id

	importCfg := testAccProviderConfig() + `
resource "devzero_cluster" "test" {
  name = "acc-cluster"
}
`
	renamedCfg := testAccProviderConfig() + `
resource "devzero_cluster" "test" {
  name = "acc-cluster-renamed"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:        importCfg,
				ResourceName:  "devzero_cluster.test",
				ImportState:   true,
				ImportStateId: clusterID,
				// Without this, the imported state is discarded at the end
				// of the step and step 2 below would silently check a plan
				// against no state at all rather than the real post-import
				// state.
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					if got := states[0].Attributes["name"]; got != "acc-cluster" {
						return fmt.Errorf("imported name = %q, want %q", got, "acc-cluster")
					}
					if got := states[0].Attributes["id"]; got != clusterID {
						return fmt.Errorf("imported id = %q, want %q", got, clusterID)
					}
					return nil
				},
			},
			{
				// Same config as the import step, applied fresh right
				// after: this must produce zero changes, proving
				// `terraform plan` right after `terraform import` is clean.
				Config:   importCfg,
				PlanOnly: true,
			},
			{
				Config: renamedCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("devzero_cluster.test", "name", "acc-cluster-renamed"),
					testAccCheckClusterNameInBackend(fake, "devzero_cluster.test", "acc-cluster-renamed"),
				),
			},
		},
	})
}

// testAccCheckClusterNameInBackend asserts the fake backend's own record was
// actually updated, independent of what Terraform's state reports.
func testAccCheckClusterNameInBackend(fake *fakeBackend, resourceName, wantName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		id := rs.Primary.ID

		fake.mu.Lock()
		defer fake.mu.Unlock()
		c, ok := fake.clusters[id]
		if !ok {
			return fmt.Errorf("cluster %s not found in fake backend", id)
		}
		if c.CustomName != wantName {
			return fmt.Errorf("fake backend cluster %s has name %q, want %q", id, c.CustomName, wantName)
		}
		return nil
	}
}
