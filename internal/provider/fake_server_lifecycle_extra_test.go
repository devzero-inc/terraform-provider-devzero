package provider

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// ---------------------------------------------------------------------------
// Cluster RPCs on the fake backend (K8SService + ClusterMutationService).
// ---------------------------------------------------------------------------

func (f *fakeBackend) CreateCluster(_ context.Context, req *connect.Request[apiv1.CreateClusterRequest]) (*connect.Response[apiv1.CreateClusterResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.clusters == nil {
		f.clusters = map[string]*apiv1.Cluster{}
	}
	c := &apiv1.Cluster{Id: f.id("cl"), Name: req.Msg.ClusterName, CustomName: req.Msg.ClusterName}
	f.clusters[c.Id] = c
	return connect.NewResponse(&apiv1.CreateClusterResponse{Cluster: c, Token: "tok-" + c.Id}), nil
}

func (f *fakeBackend) GetCluster(_ context.Context, req *connect.Request[apiv1.GetClusterRequest]) (*connect.Response[apiv1.GetClusterResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.clusters[req.Msg.ClusterId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("cluster not found"))
	}
	return connect.NewResponse(&apiv1.GetClusterResponse{Cluster: c}), nil
}

func (f *fakeBackend) UpdateCluster(_ context.Context, req *connect.Request[apiv1.UpdateClusterRequest]) (*connect.Response[apiv1.UpdateClusterResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.clusters[req.Msg.ClusterId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("cluster not found"))
	}
	c.CustomName = req.Msg.ClusterName
	return connect.NewResponse(&apiv1.UpdateClusterResponse{Cluster: c}), nil
}

func (f *fakeBackend) DeleteCluster(_ context.Context, req *connect.Request[apiv1.DeleteClusterRequest]) (*connect.Response[apiv1.DeleteClusterResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.clusters[req.Msg.ClusterId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("cluster not found"))
	}
	delete(f.clusters, req.Msg.ClusterId)
	return connect.NewResponse(&apiv1.DeleteClusterResponse{}), nil
}

func (f *fakeBackend) ResetClusterToken(_ context.Context, req *connect.Request[apiv1.ResetClusterTokenRequest]) (*connect.Response[apiv1.ResetClusterTokenResponse], error) {
	return connect.NewResponse(&apiv1.ResetClusterTokenResponse{Token: "rotated-" + req.Msg.ClusterId}), nil
}

// ---------------------------------------------------------------------------
// Cluster lifecycle
// ---------------------------------------------------------------------------

func TestLifecycle_Cluster(t *testing.T) {
	ctx := context.Background()
	fake := newFakeClientSetT(t)
	r := &ClusterResource{client: fake.cs}

	// Create
	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[ClusterResourceModel](t, r, func(m *ClusterResourceModel) {
		m.Name = stringVal("prod-cluster")
	})}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)

	var created ClusterResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))
	if created.Id.ValueString() == "" || created.Token.ValueString() == "" {
		t.Fatalf("expected id and token after create, got %+v", created)
	}

	// Update (rename) — token must be preserved, name read back with fallback.
	updResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{
		Plan: buildPlan[ClusterResourceModel](t, r, func(m *ClusterResourceModel) {
			m.Id = created.Id
			m.Name = stringVal("prod-cluster-renamed")
			m.Token = created.Token
		}),
		State: createResp.State,
	}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)
	var updated ClusterResourceModel
	mustNoDiags(t, "State.Get", updResp.State.Get(ctx, &updated))
	if updated.Name.ValueString() != "prod-cluster-renamed" {
		t.Fatalf("Name after update: %q", updated.Name.ValueString())
	}
	if updated.Token.ValueString() != created.Token.ValueString() {
		t.Fatal("token must be preserved across updates")
	}

	// Out-of-band delete → Read drops from state.
	fake.backend.mu.Lock()
	delete(fake.backend.clusters, created.Id.ValueString())
	fake.backend.mu.Unlock()
	readGone := resource.ReadResponse{State: updResp.State}
	r.Read(ctx, resource.ReadRequest{State: updResp.State}, &readGone)
	mustNoDiags(t, "Read after out-of-band delete", readGone.Diagnostics)
	if !readGone.State.Raw.IsNull() {
		t.Fatal("expected state removal after out-of-band delete")
	}

	// Delete of an already-gone cluster is tolerated.
	delResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updResp.State}, &delResp)
	mustNoDiags(t, "Delete (already gone)", delResp.Diagnostics)
}

// ---------------------------------------------------------------------------
// Update lifecycles previously untested end-to-end
// ---------------------------------------------------------------------------

func TestLifecycle_NodePolicy_Update(t *testing.T) {
	ctx := context.Background()
	fake := newFakeClientSetT(t)
	r := &NodePolicyResource{client: fake.cs}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[NodePolicyResourceModel](t, r, func(m *NodePolicyResourceModel) {
		m.Name = stringVal("pool")
		azure := &AzureNodeClass{}
		hydrateNested(t, r, "azure", azure)
		azure.VnetSubnetId = stringVal("subnet-a")
		m.Azure = azure
	})}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)
	var created NodePolicyResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))

	// Update: change the subnet and add a startup taint.
	updResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{
		Plan: buildPlan[NodePolicyResourceModel](t, r, func(m *NodePolicyResourceModel) {
			m.Id = created.Id
			m.Name = stringVal("pool")
			azure := &AzureNodeClass{}
			hydrateNested(t, r, "azure", azure)
			azure.VnetSubnetId = stringVal("subnet-b")
			m.Azure = azure
		}),
		State: createResp.State,
	}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)
	var updated NodePolicyResourceModel
	mustNoDiags(t, "State.Get", updResp.State.Get(ctx, &updated))
	if updated.Azure == nil || updated.Azure.VnetSubnetId.ValueString() != "subnet-b" {
		t.Fatalf("azure.vnet_subnet_id after update: %+v", updated.Azure)
	}
	fake.backend.mu.Lock()
	stored := fake.backend.nodePol[created.Id.ValueString()]
	fake.backend.mu.Unlock()
	if stored == nil || stored.Azure.GetVnetSubnetId() != "subnet-b" {
		t.Fatalf("backend did not receive the updated subnet: %+v", stored)
	}
}

func TestLifecycle_WorkloadPolicyTarget_Update(t *testing.T) {
	ctx := context.Background()
	fake := newFakeClientSetT(t)
	r := &WorkloadPolicyTargetResource{client: fake.cs}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[WorkloadPolicyTargetResourceModel](t, r, func(m *WorkloadPolicyTargetResourceModel) {
		m.Name = stringVal("t1")
		m.PolicyId = stringVal("wp-0001")
		m.Enabled = boolValTrue()
		m.ClusterIds = stringList("c1")
	})}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)
	var created WorkloadPolicyTargetResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))

	updResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{
		Plan: buildPlan[WorkloadPolicyTargetResourceModel](t, r, func(m *WorkloadPolicyTargetResourceModel) {
			m.Id = created.Id
			m.Name = stringVal("t1-renamed")
			m.PolicyId = stringVal("wp-0001")
			m.Enabled = boolValFalse()
			m.ClusterIds = stringList("c1")
			m.WorkloadNamesNotIn = stringList("excluded-app")
		}),
		State: createResp.State,
	}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)

	fake.backend.mu.Lock()
	stored := fake.backend.wpt[created.Id.ValueString()]
	fake.backend.mu.Unlock()
	if stored == nil {
		t.Fatal("target missing backend-side")
	}
	if stored.Name != "t1-renamed" || stored.Enabled || len(stored.WorkloadNamesNotIn) != 1 {
		t.Fatalf("update not propagated: name=%q enabled=%v notIn=%v", stored.Name, stored.Enabled, stored.WorkloadNamesNotIn)
	}
}

// ---------------------------------------------------------------------------
// Import-shaped Read: state contains only the ID (what ImportState produces).
// The Read must populate the full state from the API alone.
// ---------------------------------------------------------------------------

func TestLifecycle_WorkloadPolicy_ImportShapedRead(t *testing.T) {
	ctx := context.Background()
	fake := newFakeClientSetT(t)
	r := &WorkloadPolicyResource{client: fake.cs}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[WorkloadPolicyResourceModel](t, r, func(m *WorkloadPolicyResourceModel) {
		m.Name = stringVal("imported")
		m.EnableInPlaceVerticalScaling = boolValTrue()
		cpu := &VerticalScalingOptions{}
		hydrateNested(t, r, "cpu_vertical_scaling", cpu)
		cpu.Enabled = boolValTrue()
		m.CPUVerticalScaling = cpu
	})}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)
	var created WorkloadPolicyResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))

	// Simulate `terraform import <id>`: state holds only the id.
	importState := emptyState(t, r)
	var blank WorkloadPolicyResourceModel
	mustNoDiags(t, "hydrate blank", buildPlan[WorkloadPolicyResourceModel](t, r, func(m *WorkloadPolicyResourceModel) {
		m.Id = created.Id
	}).Get(ctx, &blank))
	mustNoDiags(t, "seed import state", importState.Set(ctx, &blank))

	readResp := resource.ReadResponse{State: importState}
	r.Read(ctx, resource.ReadRequest{State: importState}, &readResp)
	mustNoDiags(t, "Read (import-shaped)", readResp.Diagnostics)

	var imported WorkloadPolicyResourceModel
	mustNoDiags(t, "State.Get", readResp.State.Get(ctx, &imported))
	if imported.Name.ValueString() != "imported" {
		t.Fatalf("import Read did not populate name: %q", imported.Name.ValueString())
	}
	if imported.CPUVerticalScaling == nil || !imported.CPUVerticalScaling.Enabled.ValueBool() {
		t.Fatalf("import Read did not populate the nested vertical scaling block: %+v", imported.CPUVerticalScaling)
	}
	if !imported.EnableInPlaceVerticalScaling.ValueBool() {
		t.Fatal("import Read did not populate enable_in_place_vertical_scaling")
	}
}

// ---------------------------------------------------------------------------
// small typed-value helpers for this file
// ---------------------------------------------------------------------------

type fakeHarness struct {
	cs      *ClientSet
	backend *fakeBackend
}

func newFakeClientSetT(t *testing.T) fakeHarness {
	cs, backend := newFakeClientSet(t)
	return fakeHarness{cs: cs, backend: backend}
}

func stringVal(s string) types.String { return types.StringValue(s) }

func stringList(items ...string) types.List {
	vals := make([]attr.Value, 0, len(items))
	for _, it := range items {
		vals = append(vals, types.StringValue(it))
	}
	return types.ListValueMust(types.StringType, vals)
}

func boolValTrue() types.Bool  { return types.BoolValue(true) }
func boolValFalse() types.Bool { return types.BoolValue(false) }
