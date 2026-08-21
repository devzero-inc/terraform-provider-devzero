package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
	apiv1connect "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1/apiv1connect"
)

// ---------------------------------------------------------------------------
// Fake backend: an in-memory implementation of the RPCs the provider calls,
// mimicking the dakr semantics that matter to CRUD correctness (upserts,
// NotFound codes, target invariants, disabled-on-update rejection, virtual
// cluster-sourced node policies in list responses).
// ---------------------------------------------------------------------------

type fakeBackend struct {
	apiv1connect.UnimplementedK8SRecommendationServiceHandler
	apiv1connect.UnimplementedK8SServiceHandler
	apiv1connect.UnimplementedClusterMutationServiceHandler

	mu       sync.Mutex
	nextID   int
	teamID   string
	nodePol  map[string]*apiv1.NodePolicy
	nodeTgt  map[string]*apiv1.NodePolicyTarget
	wp       map[string]*apiv1.WorkloadRecommendationPolicy
	wpt      map[string]*apiv1.WorkloadPolicyTarget
	rules    map[string]*apiv1.WorkloadRule
	clusters map[string]*apiv1.Cluster
}

func newFakeBackend(teamID string) *fakeBackend {
	return &fakeBackend{
		teamID:   teamID,
		nodePol:  map[string]*apiv1.NodePolicy{},
		nodeTgt:  map[string]*apiv1.NodePolicyTarget{},
		wp:       map[string]*apiv1.WorkloadRecommendationPolicy{},
		wpt:      map[string]*apiv1.WorkloadPolicyTarget{},
		rules:    map[string]*apiv1.WorkloadRule{},
		clusters: map[string]*apiv1.Cluster{},
	}
}

func (f *fakeBackend) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s-%04d", prefix, f.nextID)
}

// --- node policies ---

func (f *fakeBackend) CreateNodePolicies(_ context.Context, req *connect.Request[apiv1.CreateNodePoliciesRequest]) (*connect.Response[apiv1.CreateNodePoliciesResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*apiv1.NodePolicy, 0, len(req.Msg.Policies))
	for _, p := range req.Msg.Policies {
		if p.Id == "" {
			p.Id = f.id("np")
		}
		p.TeamId = req.Msg.TeamId
		f.nodePol[p.Id] = p
		out = append(out, p)
	}
	return connect.NewResponse(&apiv1.CreateNodePoliciesResponse{Policies: out}), nil
}

func (f *fakeBackend) ListNodePolicies(_ context.Context, _ *connect.Request[apiv1.ListNodePoliciesRequest]) (*connect.Response[apiv1.ListNodePoliciesResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	resp := &apiv1.ListNodePoliciesResponse{}
	for _, p := range f.nodePol {
		resp.Policies = append(resp.Policies, p)
	}
	// dakr mixes read-only virtual policies mirrored from non-dakr Karpenter
	// resources into the list; the provider must skip them.
	resp.Policies = append(resp.Policies, &apiv1.NodePolicy{
		Id:     "np-virtual-cluster",
		Name:   "cluster-managed",
		Source: "cluster",
	})
	return connect.NewResponse(resp), nil
}

func (f *fakeBackend) UpdateNodePolicy(_ context.Context, req *connect.Request[apiv1.UpdateNodePolicyRequest]) (*connect.Response[apiv1.UpdateNodePolicyResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := req.Msg.Policy
	if _, ok := f.nodePol[p.Id]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("node policy not found"))
	}
	f.nodePol[p.Id] = p
	return connect.NewResponse(&apiv1.UpdateNodePolicyResponse{Policy: p}), nil
}

func (f *fakeBackend) DeleteNodePolicy(_ context.Context, req *connect.Request[apiv1.DeleteNodePolicyRequest]) (*connect.Response[apiv1.DeleteNodePolicyResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.nodePol[req.Msg.PolicyId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("resource not found"))
	}
	delete(f.nodePol, req.Msg.PolicyId)
	var deleted int32
	for id, t := range f.nodeTgt {
		if t.PolicyId == req.Msg.PolicyId {
			delete(f.nodeTgt, id)
			deleted++
		}
	}
	return connect.NewResponse(&apiv1.DeleteNodePolicyResponse{Success: true, DeletedTargetCount: deleted}), nil
}

// --- node policy targets ---

func (f *fakeBackend) CreateNodePolicyTargets(_ context.Context, req *connect.Request[apiv1.CreateNodePolicyTargetsRequest]) (*connect.Response[apiv1.CreateNodePolicyTargetsResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*apiv1.NodePolicyTarget, 0, len(req.Msg.Targets))
	for _, t := range req.Msg.Targets {
		if t.TargetId != "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target_id must be empty on create"))
		}
		if len(t.ClusterIds) > 1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("node policy target must target at most one cluster"))
		}
		t.TargetId = f.id("npt")
		f.nodeTgt[t.TargetId] = t
		out = append(out, t)
	}
	return connect.NewResponse(&apiv1.CreateNodePolicyTargetsResponse{Targets: out}), nil
}

func (f *fakeBackend) ListNodePolicyTargets(_ context.Context, _ *connect.Request[apiv1.ListNodePolicyTargetsRequest]) (*connect.Response[apiv1.ListNodePolicyTargetsResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	resp := &apiv1.ListNodePolicyTargetsResponse{}
	for _, t := range f.nodeTgt {
		resp.Targets = append(resp.Targets, t)
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeBackend) UpdateNodePolicyTarget(_ context.Context, req *connect.Request[apiv1.UpdateNodePolicyTargetRequest]) (*connect.Response[apiv1.UpdateNodePolicyTargetResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := req.Msg.Target
	if _, ok := f.nodeTgt[t.TargetId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("target not found"))
	}
	if len(t.ClusterIds) > 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("node policy target must target at most one cluster"))
	}
	f.nodeTgt[t.TargetId] = t
	return connect.NewResponse(&apiv1.UpdateNodePolicyTargetResponse{Target: t}), nil
}

// --- workload policies (v1) ---

func (f *fakeBackend) CreateWorkloadRecommendationPolicy(_ context.Context, req *connect.Request[apiv1.CreateWorkloadRecommendationPolicyRequest]) (*connect.Response[apiv1.CreateWorkloadRecommendationPolicyResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := req.Msg.Policy
	if p == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("policy required"))
	}
	p.PolicyId = f.id("wp")
	p.TeamId = req.Msg.TeamId
	f.wp[p.PolicyId] = p
	return connect.NewResponse(&apiv1.CreateWorkloadRecommendationPolicyResponse{Policy: p}), nil
}

func (f *fakeBackend) GetWorkloadRecommendationPolicy(_ context.Context, req *connect.Request[apiv1.GetWorkloadRecommendationPolicyRequest]) (*connect.Response[apiv1.GetWorkloadRecommendationPolicyResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.wp[req.Msg.PolicyId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workload policy not found"))
	}
	return connect.NewResponse(&apiv1.GetWorkloadRecommendationPolicyResponse{Policy: p}), nil
}

func (f *fakeBackend) UpdateWorkloadRecommendationPolicy(_ context.Context, req *connect.Request[apiv1.UpdateWorkloadRecommendationPolicyRequest]) (*connect.Response[apiv1.UpdateWorkloadRecommendationPolicyResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := req.Msg.Policy
	if p == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("policy required"))
	}
	if _, ok := f.wp[p.PolicyId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workload policy not found"))
	}
	f.wp[p.PolicyId] = p
	return connect.NewResponse(&apiv1.UpdateWorkloadRecommendationPolicyResponse{Policy: p}), nil
}

func (f *fakeBackend) DeleteWorkloadRecommendationPolicy(_ context.Context, req *connect.Request[apiv1.DeleteWorkloadRecommendationPolicyRequest]) (*connect.Response[apiv1.DeleteWorkloadRecommendationPolicyResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.wp[req.Msg.PolicyId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workload policy not found"))
	}
	delete(f.wp, req.Msg.PolicyId)
	return connect.NewResponse(&apiv1.DeleteWorkloadRecommendationPolicyResponse{Success: true}), nil
}

// --- workload policy targets ---

func (f *fakeBackend) CreateWorkloadPolicyTarget(_ context.Context, req *connect.Request[apiv1.CreateWorkloadPolicyTargetRequest]) (*connect.Response[apiv1.CreateWorkloadPolicyTargetResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(req.Msg.ClusterIds) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cluster_ids must name at least one cluster"))
	}
	t := &apiv1.WorkloadPolicyTarget{
		TargetId:           f.id("wpt"),
		PolicyId:           req.Msg.PolicyId,
		TeamId:             req.Msg.TeamId,
		Name:               req.Msg.Name,
		Description:        req.Msg.Description,
		Priority:           req.Msg.Priority,
		Enabled:            req.Msg.Enabled,
		NamespaceSelector:  req.Msg.NamespaceSelector,
		WorkloadSelector:   req.Msg.WorkloadSelector,
		AnnotationSelector: req.Msg.AnnotationSelector,
		KindFilter:         req.Msg.KindFilter,
		KindFilterNotIn:    req.Msg.KindFilterNotIn,
		NamePattern:        req.Msg.NamePattern,
		NamespacePattern:   req.Msg.NamespacePattern,
		WorkloadNames:      req.Msg.WorkloadNames,
		WorkloadNamesNotIn: req.Msg.WorkloadNamesNotIn,
		NodeGroupNames:     req.Msg.NodeGroupNames, //nolint:staticcheck // deprecated upstream but still round-tripped
		ClusterIds:         req.Msg.ClusterIds,
	}
	f.wpt[t.TargetId] = t
	return connect.NewResponse(&apiv1.CreateWorkloadPolicyTargetResponse{Target: t}), nil
}

func (f *fakeBackend) GetWorkloadPolicyTarget(_ context.Context, req *connect.Request[apiv1.GetWorkloadPolicyTargetRequest]) (*connect.Response[apiv1.GetWorkloadPolicyTargetResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.wpt[req.Msg.TargetId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("target not found"))
	}
	return connect.NewResponse(&apiv1.GetWorkloadPolicyTargetResponse{Target: t}), nil
}

func (f *fakeBackend) UpdateWorkloadPolicyTarget(_ context.Context, req *connect.Request[apiv1.UpdateWorkloadPolicyTargetRequest]) (*connect.Response[apiv1.UpdateWorkloadPolicyTargetResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.wpt[req.Msg.TargetId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("target not found"))
	}
	t.Name = req.Msg.Name
	t.Description = req.Msg.Description
	t.Priority = req.Msg.Priority
	t.Enabled = req.Msg.Enabled
	t.NamespaceSelector = req.Msg.NamespaceSelector
	t.WorkloadSelector = req.Msg.WorkloadSelector
	t.AnnotationSelector = req.Msg.AnnotationSelector
	t.KindFilter = req.Msg.KindFilter
	t.KindFilterNotIn = req.Msg.KindFilterNotIn
	t.NamePattern = req.Msg.NamePattern
	t.NamespacePattern = req.Msg.NamespacePattern
	t.WorkloadNames = req.Msg.WorkloadNames
	t.WorkloadNamesNotIn = req.Msg.WorkloadNamesNotIn
	t.NodeGroupNames = req.Msg.NodeGroupNames //nolint:staticcheck // deprecated upstream but still round-tripped
	if len(req.Msg.ClusterIds) > 0 {
		t.ClusterIds = req.Msg.ClusterIds
	}
	return connect.NewResponse(&apiv1.UpdateWorkloadPolicyTargetResponse{Target: t}), nil
}

func (f *fakeBackend) DeleteWorkloadPolicyTarget(_ context.Context, req *connect.Request[apiv1.DeleteWorkloadPolicyTargetRequest]) (*connect.Response[apiv1.DeleteWorkloadPolicyTargetResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	found := false
	for _, id := range req.Msg.TargetIds {
		if _, ok := f.wpt[id]; ok {
			delete(f.wpt, id)
			found = true
		}
	}
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no workload policy targets found for deletion"))
	}
	return connect.NewResponse(&apiv1.DeleteWorkloadPolicyTargetResponse{}), nil
}

// --- workload rules (v2) ---

func (f *fakeBackend) UpsertManualWorkloadRule(_ context.Context, req *connect.Request[apiv1.UpsertManualWorkloadRuleRequest]) (*connect.Response[apiv1.UpsertManualWorkloadRuleResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := req.Msg.ClusterId + "/" + req.Msg.Namespace + "/" + req.Msg.Kind + "/" + req.Msg.Name
	var existing *apiv1.WorkloadRule
	for _, r := range f.rules {
		if r.ClusterId+"/"+r.Namespace+"/"+r.Kind+"/"+r.Name == key {
			existing = r
			break
		}
	}
	if existing != nil && req.Msg.Fields != nil && req.Msg.Fields.Disabled != nil {
		// dakr: fields.disabled cannot be set when updating an existing rule.
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("fields.disabled cannot be set when updating an existing rule; use ToggleWorkloadRuleDisabled instead"))
	}
	rule := &apiv1.WorkloadRule{
		ClusterId:     req.Msg.ClusterId,
		Namespace:     req.Msg.Namespace,
		Kind:          req.Msg.Kind,
		Name:          req.Msg.Name,
		CurrentSource: workloadRuleSourceString(req.Msg.Source),
		Status:        "active",
		SyncStatus:    "pending",
		Generation:    1,
	}
	if existing != nil {
		rule.RuleId = existing.RuleId
		rule.Generation = existing.Generation + 1
		rule.Disabled = existing.Disabled
	} else {
		rule.RuleId = f.id("wr")
	}
	if fields := req.Msg.Fields; fields != nil {
		rule.CpuRule = fields.CpuRule
		rule.MemoryRule = fields.MemoryRule
		rule.GpuRule = fields.GpuRule
		rule.HpaRule = fields.HpaRule
		rule.EmergencyResponse = fields.EmergencyResponse
		rule.ActionTriggers = fields.ActionTriggers
		rule.DetectionTriggers = fields.DetectionTriggers
		rule.SchedulerPlugins = fields.SchedulerPlugins
		rule.LiveMigrationEnabled = fields.LiveMigrationEnabled
		rule.UseInPlaceVerticalScaling = fields.UseInPlaceVerticalScaling
		rule.Containers = fields.Containers
		rule.LookbackPeriodSeconds = fields.LookbackPeriodSeconds
		rule.StartupPeriodSeconds = fields.StartupPeriodSeconds
		rule.CronSchedule = fields.CronSchedule
		rule.CooldownMinutes = fields.CooldownMinutes
		rule.DefragmentationSchedule = fields.DefragmentationSchedule
		if fields.Disabled != nil {
			rule.Disabled = *fields.Disabled
		}
	}
	f.rules[rule.RuleId] = rule
	return connect.NewResponse(&apiv1.UpsertManualWorkloadRuleResponse{Rule: rule}), nil
}

func workloadRuleSourceString(s apiv1.WorkloadRuleSource) string {
	switch s {
	case apiv1.WorkloadRuleSource_WORKLOAD_RULE_SOURCE_TERRAFORM_AUTO:
		return "terraform_auto"
	case apiv1.WorkloadRuleSource_WORKLOAD_RULE_SOURCE_TERRAFORM_MANUAL:
		return "terraform_manual"
	default:
		return "manual"
	}
}

func (f *fakeBackend) GetWorkloadRuleByID(_ context.Context, req *connect.Request[apiv1.GetWorkloadRuleByIDRequest]) (*connect.Response[apiv1.GetWorkloadRuleByIDResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rules[req.Msg.RuleId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workload rule not found"))
	}
	return connect.NewResponse(&apiv1.GetWorkloadRuleByIDResponse{Rule: r}), nil
}

func (f *fakeBackend) DeleteWorkloadRule(_ context.Context, req *connect.Request[apiv1.DeleteWorkloadRuleRequest]) (*connect.Response[apiv1.DeleteWorkloadRuleResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.rules[req.Msg.RuleId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workload rule not found"))
	}
	delete(f.rules, req.Msg.RuleId)
	return connect.NewResponse(&apiv1.DeleteWorkloadRuleResponse{}), nil
}

func (f *fakeBackend) ToggleWorkloadRuleDisabled(_ context.Context, req *connect.Request[apiv1.ToggleWorkloadRuleDisabledRequest]) (*connect.Response[apiv1.ToggleWorkloadRuleDisabledResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rules[req.Msg.RuleId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workload rule not found"))
	}
	r.Disabled = req.Msg.Disabled
	return connect.NewResponse(&apiv1.ToggleWorkloadRuleDisabledResponse{Rule: r}), nil
}

// ---------------------------------------------------------------------------
// Harness plumbing
// ---------------------------------------------------------------------------

func newFakeClientSet(t *testing.T) (*ClientSet, *fakeBackend) {
	t.Helper()
	fake := newFakeBackend("team-1")
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewK8SRecommendationServiceHandler(fake))
	mux.Handle(apiv1connect.NewK8SServiceHandler(fake))
	mux.Handle(apiv1connect.NewClusterMutationServiceHandler(fake))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &ClientSet{
		TeamId:                "team-1",
		ClusterMutationClient: apiv1connect.NewClusterMutationServiceClient(srv.Client(), srv.URL),
		K8SServiceClient:      apiv1connect.NewK8SServiceClient(srv.Client(), srv.URL),
		RecommendationClient:  apiv1connect.NewK8SRecommendationServiceClient(srv.Client(), srv.URL),
	}, fake
}

// buildPlan encodes a model into a tfsdk.Plan against the resource's schema.
// The model is first hydrated from an all-null plan so every types.List/Map/
// Object field carries its proper element type (a zero types.List fails the
// framework's type verification).
func buildPlan[M any](t *testing.T, res resource.Resource, mutate func(*M)) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	nulls := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		nulls[name] = tftypes.NewValue(at, nil)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nulls)}

	var model M
	if d := plan.Get(ctx, &model); d.HasError() {
		t.Fatalf("plan.Get (hydrate): %v", d)
	}
	mutate(&model)
	if d := plan.Set(ctx, &model); d.HasError() {
		t.Fatalf("plan.Set: %v", d)
	}
	return plan
}

// hydrateNested populates target (a pointer to a nested tfsdk struct) from a
// null object of the named top-level attribute's type, so all its typed
// null fields (types.List/Map/Object) carry their element types.
func hydrateNested(t *testing.T, res resource.Resource, attrName string, target any) {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType, okObj := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !okObj {
		t.Fatal("schema type is not an object")
	}
	at, ok := objType.AttributeTypes[attrName]
	if !ok {
		t.Fatalf("attribute %q not in schema", attrName)
	}
	attrType, err := schemaResp.Schema.TypeAtTerraformPath(ctx, tftypes.NewAttributePath().WithAttributeName(attrName))
	if err != nil {
		t.Fatalf("type at path %q: %v", attrName, err)
	}
	obj, ok := at.(tftypes.Object)
	if !ok {
		t.Fatalf("attribute %q is not an object", attrName)
	}
	nulls := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, nt := range obj.AttributeTypes {
		nulls[name] = tftypes.NewValue(nt, nil)
	}
	val, err := attrType.ValueFromTerraform(ctx, tftypes.NewValue(obj, nulls))
	if err != nil {
		t.Fatalf("value from terraform: %v", err)
	}
	if d := tfsdk.ValueAs(ctx, val, target); d.HasError() {
		t.Fatalf("ValueAs: %v", d)
	}
}

func emptyState(t *testing.T, res resource.Resource) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx)
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}
}

func mustNoDiags(t *testing.T, what string, d interface{ HasError() bool }) {
	t.Helper()
	if d.HasError() {
		t.Fatalf("%s failed: %+v", what, d)
	}
}

// ---------------------------------------------------------------------------
// Lifecycle tests
// ---------------------------------------------------------------------------

func TestLifecycle_NodePolicy(t *testing.T) {
	ctx := context.Background()
	cs, fake := newFakeClientSet(t)
	r := &NodePolicyResource{client: cs}

	mutate := func(m *NodePolicyResourceModel) {
		m.Name = types.StringValue("azure-pool")
		m.Description = types.StringValue("desc")
		azure := &AzureNodeClass{}
		hydrateNested(t, r, "azure", azure)
		azure.VnetSubnetId = types.StringValue("/subscriptions/x/subnet/y")
		m.Azure = azure
	}

	// Create
	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[NodePolicyResourceModel](t, r, mutate)}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)

	var created NodePolicyResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))
	if created.Id.ValueString() == "" {
		t.Fatal("expected id after create")
	}
	if created.Azure == nil || created.Azure.VnetSubnetId.ValueString() != "/subscriptions/x/subnet/y" {
		t.Fatalf("azure block not round-tripped: %+v", created.Azure)
	}

	// Read refreshes from the API and must skip the virtual source=="cluster" policy.
	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	mustNoDiags(t, "Read", readResp.Diagnostics)

	// Out-of-band deletion drops the resource from state instead of erroring.
	fake.mu.Lock()
	delete(fake.nodePol, created.Id.ValueString())
	fake.mu.Unlock()
	readGone := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readGone)
	mustNoDiags(t, "Read after out-of-band delete", readGone.Diagnostics)
	if !readGone.State.Raw.IsNull() {
		t.Fatal("expected state to be removed after out-of-band delete")
	}

	// Recreate, then Delete must call the API (policy actually disappears) and
	// a second Delete must tolerate NotFound.
	createResp = resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[NodePolicyResourceModel](t, r, mutate)}, &createResp)
	mustNoDiags(t, "re-Create", createResp.Diagnostics)
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))

	delResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &delResp)
	mustNoDiags(t, "Delete", delResp.Diagnostics)
	fake.mu.Lock()
	_, still := fake.nodePol[created.Id.ValueString()]
	fake.mu.Unlock()
	if still {
		t.Fatal("Delete did not remove the policy from the backend")
	}
	delAgain := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &delAgain)
	mustNoDiags(t, "Delete (already gone)", delAgain.Diagnostics)
}

func TestLifecycle_NodePolicyTarget(t *testing.T) {
	ctx := context.Background()
	cs, fake := newFakeClientSet(t)
	r := &NodePolicyTargetResource{client: cs}

	mutate := func(m *NodePolicyTargetResourceModel) {
		m.Name = types.StringValue("target-1")
		m.PolicyId = types.StringValue("np-0001")
		m.Enabled = types.BoolValue(true)
		m.ClusterIds = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("cluster-1")})
	}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[NodePolicyTargetResourceModel](t, r, mutate)}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)

	var created NodePolicyTargetResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))
	if created.Id.ValueString() == "" {
		t.Fatal("expected target id")
	}

	// Destroy must disable the target (no delete RPC exists).
	delResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &delResp)
	mustNoDiags(t, "Delete", delResp.Diagnostics)
	fake.mu.Lock()
	tgt := fake.nodeTgt[created.Id.ValueString()]
	fake.mu.Unlock()
	if tgt == nil {
		t.Fatal("target should still exist backend-side (no delete RPC)")
	}
	if tgt.Enabled {
		t.Fatal("Delete must disable the target")
	}
}

func TestLifecycle_WorkloadPolicy(t *testing.T) {
	ctx := context.Background()
	cs, fake := newFakeClientSet(t)
	r := &WorkloadPolicyResource{client: cs}

	mutate := func(m *WorkloadPolicyResourceModel) {
		m.Name = types.StringValue("policy-1")
		m.EnableInPlaceVerticalScaling = types.BoolValue(true)
		cpu := &VerticalScalingOptions{}
		hydrateNested(t, r, "cpu_vertical_scaling", cpu)
		cpu.Enabled = types.BoolValue(true)
		cpu.MinRequest = types.Int64Value(10)
		m.CPUVerticalScaling = cpu
	}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[WorkloadPolicyResourceModel](t, r, mutate)}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)
	var created WorkloadPolicyResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))

	// Read after out-of-band delete removes from state.
	fake.mu.Lock()
	delete(fake.wp, created.Id.ValueString())
	fake.mu.Unlock()
	readGone := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readGone)
	mustNoDiags(t, "Read after out-of-band delete", readGone.Diagnostics)
	if !readGone.State.Raw.IsNull() {
		t.Fatal("expected state removal")
	}

	// allow_in_place_memory_limit_decrease without in-place scaling is rejected client-side.
	badResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[WorkloadPolicyResourceModel](t, r, func(m *WorkloadPolicyResourceModel) {
		m.Name = types.StringValue("bad")
		m.AllowInPlaceMemoryLimitDecrease = types.BoolValue(true)
		m.EnableInPlaceVerticalScaling = types.BoolValue(false)
	})}, &badResp)
	if !badResp.Diagnostics.HasError() {
		t.Fatal("expected validation error for allow_in_place_memory_limit_decrease without enable_in_place_vertical_scaling")
	}
}

func TestLifecycle_WorkloadRule(t *testing.T) {
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
	if created.Id.ValueString() == "" {
		t.Fatal("expected rule id")
	}

	// Update flipping `disabled` must go through the toggle RPC, not the upsert
	// (the fake rejects fields.disabled on update, mirroring dakr).
	updatePlan := buildPlan[WorkloadRuleResourceModel](t, r, func(m *WorkloadRuleResourceModel) {
		mutate(m)
		m.Id = created.Id
		m.Disabled = types.BoolValue(true)
	})
	updResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: createResp.State}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)
	fake.mu.Lock()
	rule := fake.rules[created.Id.ValueString()]
	fake.mu.Unlock()
	if rule == nil || !rule.Disabled {
		t.Fatalf("expected rule to be disabled via toggle RPC, got %+v", rule)
	}

	// Delete + idempotent delete.
	delResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updResp.State}, &delResp)
	mustNoDiags(t, "Delete", delResp.Diagnostics)
	delAgain := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updResp.State}, &delAgain)
	mustNoDiags(t, "Delete (already gone)", delAgain.Diagnostics)
}

func TestLifecycle_WorkloadPolicyTarget(t *testing.T) {
	ctx := context.Background()
	cs, _ := newFakeClientSet(t)
	r := &WorkloadPolicyTargetResource{client: cs}

	mutate := func(m *WorkloadPolicyTargetResourceModel) {
		m.Name = types.StringValue("t1")
		m.PolicyId = types.StringValue("wp-0001")
		m.Enabled = types.BoolValue(true)
		m.ClusterIds = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("c1")})
		m.NamePattern = &RegexPattern{
			Pattern: types.StringValue("^api-"),
			Flags:   types.StringNull(), // omitted: must round-trip as null, not "".
		}
	}

	createResp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[WorkloadPolicyTargetResourceModel](t, r, mutate)}, &createResp)
	mustNoDiags(t, "Create", createResp.Diagnostics)
	var created WorkloadPolicyTargetResourceModel
	mustNoDiags(t, "State.Get", createResp.State.Get(ctx, &created))
	if created.NamePattern == nil || !created.NamePattern.Flags.IsNull() {
		t.Fatalf("expected null flags after round-trip, got %+v", created.NamePattern)
	}

	delResp := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &delResp)
	mustNoDiags(t, "Delete", delResp.Diagnostics)
	delAgain := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &delAgain)
	mustNoDiags(t, "Delete (already gone)", delAgain.Diagnostics)
}
