package provider

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// ---------------------------------------------------------------------------
// devzero_cluster.team_id: plans through the provider server, as Terraform
// runs them, so the schema's plan modifiers and ModifyPlan both apply.
// ---------------------------------------------------------------------------

// clusterValue builds a devzero_cluster object; an empty string is null.
func clusterValue(t *testing.T, id, name, token, team string) tftypes.Value {
	t.Helper()
	s := func(v string) tftypes.Value {
		if v == "" {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, v)
	}
	return tftypes.NewValue(clusterObjectType(t), map[string]tftypes.Value{
		"id": s(id), "name": s(name), "token": s(token), "team_id": s(team),
	})
}

func clusterObjectType(t *testing.T) tftypes.Object {
	t.Helper()
	obj, ok := resourceSchema(t, NewClusterResource()).Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	return obj
}

func dynamicValue(t *testing.T, typ tftypes.Type, v tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	dv, err := tfprotov6.NewDynamicValue(typ, v)
	if err != nil {
		t.Fatalf("dynamic value: %v", err)
	}
	return &dv
}

// planCluster plans an update of prior against a configuration holding
// configName and configTeam ("" leaves team_id unset), with the provider
// configured for providerTeam.
func planCluster(t *testing.T, providerTeam string, prior tftypes.Value, configName, configTeam string) (tftypes.Value, []*tfprotov6.Diagnostic) {
	t.Helper()
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()

	providerType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"url": tftypes.String, "team_id": tftypes.String, "token": tftypes.String,
	}}
	confResp, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{
		TerraformVersion: "1.9.0",
		Config: dynamicValue(t, providerType, tftypes.NewValue(providerType, map[string]tftypes.Value{
			"url":     tftypes.NewValue(tftypes.String, "http://127.0.0.1:1"),
			"team_id": tftypes.NewValue(tftypes.String, providerTeam),
			"token":   tftypes.NewValue(tftypes.String, "dzs-test"),
		})),
	})
	if err != nil {
		t.Fatalf("ConfigureProvider: %v", err)
	}
	for _, d := range confResp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("ConfigureProvider: %s: %s", d.Summary, d.Detail)
		}
	}

	obj := clusterObjectType(t)
	var priorAttrs map[string]tftypes.Value
	if err := prior.As(&priorAttrs); err != nil {
		t.Fatalf("prior: %v", err)
	}
	// Terraform proposes the prior value for a computed attribute the
	// configuration leaves null.
	proposed := map[string]tftypes.Value{
		"id":      priorAttrs["id"],
		"token":   priorAttrs["token"],
		"name":    tftypes.NewValue(tftypes.String, configName),
		"team_id": priorAttrs["team_id"],
	}
	if configTeam != "" {
		proposed["team_id"] = tftypes.NewValue(tftypes.String, configTeam)
	}

	planResp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "devzero_cluster",
		PriorState:       dynamicValue(t, obj, prior),
		ProposedNewState: dynamicValue(t, obj, tftypes.NewValue(obj, proposed)),
		Config:           dynamicValue(t, obj, clusterValue(t, "", configName, "", configTeam)),
	})
	if err != nil {
		t.Fatalf("PlanResourceChange: %v", err)
	}
	if planResp.PlannedState == nil {
		return tftypes.NewValue(obj, nil), planResp.Diagnostics
	}
	planned, err := planResp.PlannedState.Unmarshal(obj)
	if err != nil {
		t.Fatalf("planned state: %v", err)
	}
	return planned, planResp.Diagnostics
}

func TestClusterPlan_TeamIdNeverMovesACluster(t *testing.T) {
	teamIDPath := tftypes.NewAttributePath().WithAttributeName("team_id")
	cases := []struct {
		name         string
		providerTeam string
		prior        tftypes.Value
		configName   string
		configTeam   string
		// wantNoDiff: the plan equals the prior state.
		wantNoDiff bool
		wantTeam   string
		wantErr    bool
	}{
		{
			// Fails if the plan defaults team_id from the provider's team.
			name: "state from today's release, not yet refreshed, plans no diff", providerTeam: "team-2",
			prior: clusterValue(t, "cl-1", "prod", "tok", ""), configName: "prod", wantNoDiff: true,
		},
		{
			// Fails if the plan defaults team_id from the provider's team.
			name: "unset team after the provider's team changed plans no diff", providerTeam: "team-2",
			prior: clusterValue(t, "cl-1", "prod", "tok", "team-1"), configName: "prod", wantNoDiff: true,
		},
		{
			// Fails if team_id loses UseStateForUnknown: a rename would plan it unknown.
			name: "a rename keeps the stored team", providerTeam: "team-2",
			prior: clusterValue(t, "cl-1", "prod", "tok", "team-1"), configName: "prod-renamed", wantTeam: "team-1",
		},
		{
			// Fails if ModifyPlan refuses a team_id equal to the stored one.
			name: "the stored team set explicitly plans no diff", providerTeam: "team-2",
			prior: clusterValue(t, "cl-1", "prod", "tok", "team-1"), configName: "prod", configTeam: "team-1", wantNoDiff: true,
		},
		{
			// Fails if ModifyPlan drops the team_id check.
			name: "a change of a known team is refused", providerTeam: "team-1",
			prior: clusterValue(t, "cl-1", "prod", "tok", "team-1"), configName: "prod", configTeam: "team-2", wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			planned, diags := planCluster(t, tc.providerTeam, tc.prior, tc.configName, tc.configTeam)

			var errAtTeam, otherErr bool
			for _, d := range diags {
				if d.Severity != tfprotov6.DiagnosticSeverityError {
					continue
				}
				if d.Attribute != nil && d.Attribute.Equal(teamIDPath) {
					errAtTeam = true
				} else {
					otherErr = true
				}
			}
			if otherErr || errAtTeam != tc.wantErr {
				t.Fatalf("diagnostics: %+v, want an error at team_id: %v", diags, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if tc.wantNoDiff && !planned.Equal(tc.prior) {
				t.Fatalf("planned %v, want the prior state %v", planned, tc.prior)
			}
			if tc.wantTeam != "" {
				var attrs map[string]tftypes.Value
				if err := planned.As(&attrs); err != nil {
					t.Fatalf("planned: %v", err)
				}
				if !attrs["team_id"].Equal(tftypes.NewValue(tftypes.String, tc.wantTeam)) {
					t.Fatalf("planned team_id %v, want %q", attrs["team_id"], tc.wantTeam)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Calls against the fake, which answers NotFound to a call naming a team the
// cluster is not in.
// ---------------------------------------------------------------------------

func clusterState(t *testing.T, r *ClusterResource, mutate func(*ClusterResourceModel)) tfsdk.State {
	t.Helper()
	p := buildPlan[ClusterResourceModel](t, r, mutate)
	return tfsdk.State{Schema: p.Schema, Raw: p.Raw}
}

// seedCluster stores a cluster in team on the fake and returns its id.
func seedCluster(h fakeHarness, team string) string {
	h.backend.mu.Lock()
	defer h.backend.mu.Unlock()
	c := &apiv1.Cluster{Id: h.backend.id("cl"), Name: "prod", CustomName: "prod"}
	h.backend.clusters[c.Id] = c
	h.backend.clusterTeam[c.Id] = team
	return c.Id
}

func TestClusterRead_TeamIdTakesTheProvidersTeamOnlyWhenUnset(t *testing.T) {
	cases := []struct {
		name         string
		clusterTeam  string
		stateTeam    types.String
		wantTeam     string
		providerTeam string
	}{
		// Fails if Read leaves a null team_id unset.
		{name: "state from today's release takes the provider's team", clusterTeam: "team-1", stateTeam: types.StringNull(), providerTeam: "team-1", wantTeam: "team-1"},
		// Fails if Read calls with the provider's team instead of the stored one.
		{name: "a stored team stays when the provider's team changes", clusterTeam: "team-1", stateTeam: stringVal("team-1"), providerTeam: "team-2", wantTeam: "team-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			h := newFakeClientSetT(t)
			h.cs.TeamId = tc.providerTeam
			r := &ClusterResource{client: h.cs}
			id := seedCluster(h, tc.clusterTeam)
			state := clusterState(t, r, func(m *ClusterResourceModel) {
				m.Id = stringVal(id)
				m.Name = stringVal("prod")
				m.Token = stringVal("tok")
				m.TeamId = tc.stateTeam
			})

			readResp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
			mustNoDiags(t, "Read", readResp.Diagnostics)
			if readResp.State.Raw.IsNull() {
				t.Fatal("Read dropped the cluster from state")
			}
			var got ClusterResourceModel
			mustNoDiags(t, "State.Get", readResp.State.Get(ctx, &got))
			if got.TeamId.ValueString() != tc.wantTeam {
				t.Fatalf("team_id after Read: %q, want %q", got.TeamId.ValueString(), tc.wantTeam)
			}
		})
	}
}

func TestClusterRead_FailsWithoutTouchingStateOutsideTheProvidersTeam(t *testing.T) {
	cases := []struct {
		name       string
		stateTeam  string
		getErr     error
		wantDetail string
	}{
		// Fails if Read drops a cluster of another team from state, as it does in the provider's own team.
		{name: "a cluster of another team is gone", stateTeam: "team-2", wantDetail: "removed blocks"},
		// Fails if Read drops the cluster or swallows an error other than NotFound.
		{name: "an error other than NotFound", stateTeam: "team-1", getErr: connect.NewError(connect.CodeUnavailable, errors.New("dakr is restarting")), wantDetail: "dakr is restarting"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			h := newFakeClientSetT(t)
			h.backend.getClusterErr = tc.getErr
			r := &ClusterResource{client: h.cs}
			state := clusterState(t, r, func(m *ClusterResourceModel) {
				m.Id = stringVal("cl-gone")
				m.Name = stringVal("prod")
				m.Token = stringVal("tok")
				m.TeamId = stringVal(tc.stateTeam)
			})

			readResp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
			if !hasDiag(readResp.Diagnostics, diag.SeverityError, tc.wantDetail) {
				t.Fatalf("diagnostics %+v, want an error containing %q", readResp.Diagnostics, tc.wantDetail)
			}
			if !readResp.State.Raw.Equal(state.Raw) {
				t.Fatal("Read changed the state on a failed refresh")
			}
		})
	}
}

// Fails if Create, Read, Update or Delete call with the provider's team
// instead of the cluster's: the fake answers NotFound in another team.
func TestCluster_EveryCallGoesToTheClustersTeam(t *testing.T) {
	ctx := context.Background()
	h := newFakeClientSetT(t)
	r := &ClusterResource{client: h.cs}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[ClusterResourceModel](t, r, func(m *ClusterResourceModel) {
		m.Name = stringVal("acme-prod")
		m.TeamId = stringVal("team-acme")
	})}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)
	var created ClusterResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))
	h.backend.mu.Lock()
	team := h.backend.clusterTeam[created.Id.ValueString()]
	h.backend.mu.Unlock()
	if team != "team-acme" || created.TeamId.ValueString() != "team-acme" {
		t.Fatalf("cluster registered in %q with team_id %q, want team-acme", team, created.TeamId.ValueString())
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	mustNoDiags(t, "Read", readResp.Diagnostics)

	updResp := resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{
		Plan: buildPlan[ClusterResourceModel](t, r, func(m *ClusterResourceModel) {
			m.Id = created.Id
			m.Name = stringVal("acme-prod-renamed")
			m.Token = created.Token
			m.TeamId = stringVal("team-acme")
		}),
		State: readResp.State,
	}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)

	delResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updResp.State}, &delResp)
	mustNoDiags(t, "Delete", delResp.Diagnostics)
	h.backend.mu.Lock()
	_, still := h.backend.clusters[created.Id.ValueString()]
	h.backend.mu.Unlock()
	if still {
		t.Fatal("Delete did not reach the cluster in its team")
	}
}

// Fails if Update sends the unknown team_id as an empty team instead of the
// provider's: the fake answers NotFound.
func TestClusterUpdate_UnknownTeamSendsTheProvidersTeam(t *testing.T) {
	ctx := context.Background()
	h := newFakeClientSetT(t)
	r := &ClusterResource{client: h.cs}
	id := seedCluster(h, h.cs.TeamId)
	state := clusterState(t, r, func(m *ClusterResourceModel) {
		m.Id = stringVal(id)
		m.Name = stringVal("prod")
		m.Token = stringVal("tok")
	})

	updResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{
		Plan: buildPlan[ClusterResourceModel](t, r, func(m *ClusterResourceModel) {
			m.Id = stringVal(id)
			m.Name = stringVal("prod-renamed")
			m.Token = stringVal("tok")
			m.TeamId = types.StringUnknown()
		}),
		State: state,
	}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)
	var got ClusterResourceModel
	mustNoDiags(t, "State.Get", updResp.State.Get(ctx, &got))
	if got.TeamId.ValueString() != h.cs.TeamId {
		t.Fatalf("team_id after Update: %q, want the provider's %q", got.TeamId.ValueString(), h.cs.TeamId)
	}
	h.backend.mu.Lock()
	name := h.backend.clusters[id].CustomName
	h.backend.mu.Unlock()
	if name != "prod-renamed" {
		t.Fatalf("rename did not reach the cluster: %q", name)
	}
}

// Fails if ImportState passes `<team_id>/<cluster_id>` through as the id.
func TestClusterImport_TakesAPlainIdOrTeamSlashId(t *testing.T) {
	cases := []struct {
		id       string
		wantID   string
		wantTeam types.String
		wantErr  bool
	}{
		{id: "team-2/cl-9", wantID: "cl-9", wantTeam: stringVal("team-2")},
		{id: "cl-9", wantID: "cl-9", wantTeam: types.StringNull()},
		{id: "team-2/", wantErr: true},
		{id: "/cl-9", wantErr: true},
		{id: "team-2/cl-9/x", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			ctx := context.Background()
			r := &ClusterResource{}
			resp := resource.ImportStateResponse{State: emptyState(t, r)}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.id}, &resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("diagnostics %+v, want an error: %v", resp.Diagnostics, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			var id, team types.String
			mustNoDiags(t, "id", resp.State.GetAttribute(ctx, path.Root("id"), &id))
			mustNoDiags(t, "team_id", resp.State.GetAttribute(ctx, path.Root("team_id"), &team))
			if id.ValueString() != tc.wantID || !team.Equal(tc.wantTeam) {
				t.Fatalf("imported id %v team_id %v, want %q and %v", id, team, tc.wantID, tc.wantTeam)
			}
		})
	}
}
