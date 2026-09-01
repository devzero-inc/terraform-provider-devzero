package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	apiv1connect "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1/apiv1connect"
)

// ---------------------------------------------------------------------------
// Acceptance-test harness: wires the real provider (via ProtoV6ProviderFactories)
// against the same in-memory fake backend the unit tests use, so
// resource.TestCase can drive genuine `terraform import` / `terraform plan` /
// `terraform apply` cycles without hitting a real DevZero API.
// ---------------------------------------------------------------------------

const (
	testAccTeamID = "team-1"
	testAccToken  = "test-token"
)

// testAccHarness starts a fake backend and returns the provider factories
// terraform-plugin-testing needs, plus the backend for direct assertions
// against what was actually persisted (independent of what Terraform reports).
func testAccHarness(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeBackend) {
	t.Helper()

	fake := newFakeBackend(testAccTeamID)
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewK8SRecommendationServiceHandler(fake))
	mux.Handle(apiv1connect.NewK8SServiceHandler(fake))
	mux.Handle(apiv1connect.NewClusterMutationServiceHandler(fake))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"devzero": providerserver.NewProtocol6WithError(&DevzeroProvider{version: "test", defaultURL: srv.URL}),
	}
	return factories, fake
}

// testAccProviderConfig renders the provider block every test config needs;
// the URL is baked into the provider instance itself (see testAccHarness), so
// only credentials the fake backend expects are supplied here.
func testAccProviderConfig() string {
	return fmt.Sprintf(`
provider "devzero" {
  team_id = %q
  token   = %q
}
`, testAccTeamID, testAccToken)
}
