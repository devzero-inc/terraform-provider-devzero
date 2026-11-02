package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

// ---------------------------------------------------------------------------
// TeamService on the fake backend: the caller's started teams.
// ---------------------------------------------------------------------------

func (f *fakeBackend) CreateTeam(_ context.Context, req *connect.Request[apiv1.CreateTeamRequest]) (*connect.Response[apiv1.CreateTeamResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createTeamReqs = append(f.createTeamReqs, req.Msg)
	if f.createTeamErr != nil {
		return nil, f.createTeamErr
	}
	team := &apiv1.Team{Id: f.id("started"), Name: req.Msg.Name}
	f.teams[team.Id] = team
	return connect.NewResponse(&apiv1.CreateTeamResponse{Team: team, InvitesNotSent: f.invitesNotSent}), nil
}

func (f *fakeBackend) GetTeam(_ context.Context, req *connect.Request[apiv1.GetTeamRequest]) (*connect.Response[apiv1.GetTeamResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getTeamErr != nil {
		return nil, f.getTeamErr
	}
	team, ok := f.teams[req.Msg.TeamId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("team not found"))
	}
	return connect.NewResponse(&apiv1.GetTeamResponse{Team: team}), nil
}

func (f *fakeBackend) LeaveTeam(_ context.Context, req *connect.Request[apiv1.LeaveTeamRequest]) (*connect.Response[apiv1.LeaveTeamResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.leaveTeamReqs = append(f.leaveTeamReqs, req.Msg)
	if f.leaveTeamErr != nil {
		return nil, f.leaveTeamErr
	}
	if _, ok := f.teams[req.Msg.TeamId]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("resource not found"))
	}
	delete(f.teams, req.Msg.TeamId)
	return connect.NewResponse(&apiv1.LeaveTeamResponse{TeamDeleted: f.leaveDeletes}), nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func stringSet(items ...string) types.Set {
	vals := make([]attr.Value, 0, len(items))
	for _, it := range items {
		vals = append(vals, types.StringValue(it))
	}
	return types.SetValueMust(types.StringType, vals)
}

func teamState(t *testing.T, r *TeamResource, mutate func(*TeamResourceModel)) tfsdk.State {
	t.Helper()
	p := buildPlan[TeamResourceModel](t, r, mutate)
	return tfsdk.State{Schema: p.Schema, Raw: p.Raw}
}

func startAcme(m *TeamResourceModel) {
	m.Name = stringVal("Acme")
	m.MemberEmails = stringSet("alice@partner.example", "bob@partner.example")
	m.AdminInviteEmails = stringSet("admin@acme.example")
}

// startTeam creates Acme through the resource and returns its state.
func startTeam(t *testing.T, r *TeamResource) (tfsdk.State, TeamResourceModel) {
	t.Helper()
	ctx := context.Background()
	resp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(ctx, resource.CreateRequest{Plan: buildPlan[TeamResourceModel](t, r, startAcme)}, &resp)
	mustNoDiags(t, "Create", resp.Diagnostics)
	var created TeamResourceModel
	mustNoDiags(t, "State.Get", resp.State.Get(ctx, &created))
	return resp.State, created
}

func setStrings(t *testing.T, s types.Set) []string {
	t.Helper()
	var out []string
	mustNoDiags(t, "ElementsAs", s.ElementsAs(context.Background(), &out, false))
	slices.Sort(out)
	return out
}

func hasDiag(diags diag.Diagnostics, severity diag.Severity, text string) bool {
	for _, d := range diags {
		if d.Severity() == severity && strings.Contains(d.Summary()+" "+d.Detail(), text) {
			return true
		}
	}
	return false
}

func errorAt(diags diag.Diagnostics, p path.Path) bool {
	for _, d := range diags.Errors() {
		if dp, ok := d.(diag.DiagnosticWithPath); ok && dp.Path().Equal(p) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestTeamCreate_StartsTheTeamFromTheProvidersTeam(t *testing.T) {
	cases := []struct {
		name    string
		notSent []string
	}{
		// Fails if Create drops the provider's team_id or a field from the request.
		{name: "every invite emailed"},
		// Fails if Create ignores invites_not_sent or raises no warning for it.
		{name: "an invite email refused", notSent: []string{"admin@acme.example"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newFakeClientSetT(t)
			h.backend.invitesNotSent = tc.notSent
			r := &TeamResource{client: h.cs}

			ctx := context.Background()
			resp := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(ctx, resource.CreateRequest{Plan: buildPlan[TeamResourceModel](t, r, startAcme)}, &resp)
			mustNoDiags(t, "Create", resp.Diagnostics)

			h.backend.mu.Lock()
			reqs := h.backend.createTeamReqs
			h.backend.mu.Unlock()
			if len(reqs) != 1 {
				t.Fatalf("CreateTeam calls: %d, want 1", len(reqs))
			}
			got := reqs[0]
			if got.TeamId != "team-1" || got.Name != "Acme" ||
				!slices.Equal(got.MemberEmails, []string{"alice@partner.example", "bob@partner.example"}) ||
				!slices.Equal(got.AdminInviteEmails, []string{"admin@acme.example"}) {
				t.Fatalf("CreateTeam request: team_id=%q name=%q member_emails=%v admin_invite_emails=%v",
					got.TeamId, got.Name, got.MemberEmails, got.AdminInviteEmails)
			}

			var created TeamResourceModel
			mustNoDiags(t, "State.Get", resp.State.Get(ctx, &created))
			h.backend.mu.Lock()
			_, stored := h.backend.teams[created.Id.ValueString()]
			h.backend.mu.Unlock()
			if !stored {
				t.Fatalf("state id %q is not the started team", created.Id.ValueString())
			}
			if notSent := setStrings(t, created.InvitesNotSent); !slices.Equal(notSent, tc.notSent) {
				t.Fatalf("invites_not_sent: %v, want %v", notSent, tc.notSent)
			}
			warned := hasDiag(resp.Diagnostics, diag.SeverityWarning, "admin@acme.example")
			if warned != (len(tc.notSent) > 0) {
				t.Fatalf("warning naming the unsent invite: %v, want %v (%+v)", warned, len(tc.notSent) > 0, resp.Diagnostics)
			}
		})
	}
}

// Fails if Create writes state on a refusal, or drops dakr's reason from the
// diagnostic. Each row is a refusal dakr answers a start with.
func TestTeamCreate_RefusalSurfacesAsAnErrorAndWritesNoState(t *testing.T) {
	cases := []struct {
		name string
		code connect.Code
		text string
	}{
		{name: "starting team is not a partner", code: connect.CodePermissionDenied,
			text: "only a DevZero partner team can start teams or let its service accounts into other teams; contact support@devzero.io"},
		{name: "caller is a Viewer of the starting team", code: connect.CodePermissionDenied,
			text: "only an Admin or Developer of a team can start a team from it"},
		{name: "service account starts from a team not its own", code: connect.CodePermissionDenied,
			text: "a service account can start teams only from its own team"},
		{name: "member email is not an Admin or Developer there", code: connect.CodeInvalidArgument,
			text: "alice@partner.example is not a person with the Admin or Developer role in the team this one is started from"},
		{name: "admin invite to a person of the starting team", code: connect.CodeInvalidArgument,
			text: "admin@acme.example is in the team this one is started from and cannot be invited to it as Admin"},
		{name: "no person joins as Admin", code: connect.CodeInvalidArgument,
			text: "a person must join the new team as Admin"},
		{name: "blank name", code: connect.CodeInvalidArgument,
			text: "team name must have at least one character"},
		{name: "invites rate limited", code: connect.CodeResourceExhausted,
			text: "rate limit exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newFakeClientSetT(t)
			h.backend.createTeamErr = connect.NewError(tc.code, errors.New(tc.text))
			r := &TeamResource{client: h.cs}

			resp := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(context.Background(), resource.CreateRequest{Plan: buildPlan[TeamResourceModel](t, r, startAcme)}, &resp)
			if !hasDiag(resp.Diagnostics, diag.SeverityError, tc.text) {
				t.Fatalf("diagnostics %+v, want an error carrying %q", resp.Diagnostics, tc.text)
			}
			if !resp.State.Raw.IsNull() {
				t.Fatal("Create wrote state for a refused start")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Plan: the values a team is started with
// ---------------------------------------------------------------------------

func TestTeamPlan_RefusesAChangeToWhatTheTeamWasStartedWith(t *testing.T) {
	started := func(m *TeamResourceModel) {
		m.Id = stringVal("started-0001")
		m.Name = stringVal("Acme")
		m.MemberEmails = stringSet("alice@partner.example")
		m.InvitesNotSent = stringSet()
	}
	cases := []struct {
		name      string
		prior     func(*TeamResourceModel)
		planned   func(*TeamResourceModel)
		wantErrAt string
	}{
		// Fails if ModifyPlan refuses an unchanged plan.
		{name: "unchanged", prior: started, planned: started},
		// Fails if ModifyPlan drops the name check.
		{name: "renamed", prior: started, wantErrAt: "name", planned: func(m *TeamResourceModel) {
			started(m)
			m.Name = stringVal("Acme Production")
		}},
		// Fails if ModifyPlan drops the member_emails check.
		{name: "a member added", prior: started, wantErrAt: "member_emails", planned: func(m *TeamResourceModel) {
			started(m)
			m.MemberEmails = stringSet("alice@partner.example", "bob@partner.example")
		}},
		// Fails if a null prior value alone lets a change through: this team was started without admin invites.
		{name: "an admin invite added after a start without one", prior: started, wantErrAt: "admin_invite_emails", planned: func(m *TeamResourceModel) {
			started(m)
			m.AdminInviteEmails = stringSet("admin@acme.example")
		}},
		// Fails if ModifyPlan refuses the configured values of an imported team.
		{name: "an imported team takes the configured values", prior: func(m *TeamResourceModel) {
			m.Id = stringVal("started-0001")
		}, planned: func(m *TeamResourceModel) {
			startAcme(m)
			m.Id = stringVal("started-0001")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &TeamResource{}
			plan := buildPlan[TeamResourceModel](t, r, tc.planned)
			resp := resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: teamState(t, r, tc.prior), Plan: plan}, &resp)
			if tc.wantErrAt == "" {
				mustNoDiags(t, "ModifyPlan", resp.Diagnostics)
				return
			}
			if len(resp.Diagnostics.Errors()) != 1 || !errorAt(resp.Diagnostics, path.Root(tc.wantErrAt)) {
				t.Fatalf("diagnostics %+v, want one error at %s", resp.Diagnostics, tc.wantErrAt)
			}
		})
	}
}

// Fails if the import-shaped Read fails, or if the Update that follows an
// import sends anything.
func TestTeamImport_ThenApplyRecordsTheConfigurationWithoutACall(t *testing.T) {
	ctx := context.Background()
	h := newFakeClientSetT(t)
	h.backend.teams["started-0042"] = &apiv1.Team{Id: "started-0042", Name: "Acme Inc"}
	r := &TeamResource{client: h.cs}

	importResp := resource.ImportStateResponse{State: emptyState(t, r)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "started-0042"}, &importResp)
	mustNoDiags(t, "ImportState", importResp.Diagnostics)
	readResp := resource.ReadResponse{State: importResp.State}
	r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	mustNoDiags(t, "Read", readResp.Diagnostics)

	updResp := resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{
		Plan: buildPlan[TeamResourceModel](t, r, func(m *TeamResourceModel) {
			startAcme(m)
			m.Id = stringVal("started-0042")
		}),
		State: readResp.State,
	}, &updResp)
	mustNoDiags(t, "Update", updResp.Diagnostics)

	var got TeamResourceModel
	mustNoDiags(t, "State.Get", updResp.State.Get(ctx, &got))
	if got.Id.ValueString() != "started-0042" || got.Name.ValueString() != "Acme" ||
		!slices.Equal(setStrings(t, got.MemberEmails), []string{"alice@partner.example", "bob@partner.example"}) {
		t.Fatalf("state after the import's apply: %+v", got)
	}
	h.backend.mu.Lock()
	creates, leaves := len(h.backend.createTeamReqs), len(h.backend.leaveTeamReqs)
	h.backend.mu.Unlock()
	if creates != 0 || leaves != 0 {
		t.Fatalf("the import's apply called CreateTeam %d and LeaveTeam %d times, want none", creates, leaves)
	}
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

func TestTeamRead_KeepsStateAndNeverDropsTheTeam(t *testing.T) {
	cases := []struct {
		name       string
		arrange    func(h fakeHarness, id string)
		wantDetail string
	}{
		// Fails if Read writes the team's current name over the one it was started with.
		{name: "renamed by its Admins", arrange: func(h fakeHarness, id string) {
			h.backend.teams[id].Name = "Acme Inc"
		}},
		// Fails if Read drops the team from state on NotFound, which would start a new team on the next apply.
		{name: "gone", wantDetail: "removed blocks", arrange: func(h fakeHarness, id string) {
			delete(h.backend.teams, id)
		}},
		// Fails if Read drops the team or swallows an error other than NotFound.
		{name: "dakr unavailable", wantDetail: "dakr is restarting", arrange: func(h fakeHarness, _ string) {
			h.backend.getTeamErr = connect.NewError(connect.CodeUnavailable, errors.New("dakr is restarting"))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			h := newFakeClientSetT(t)
			r := &TeamResource{client: h.cs}
			state, created := startTeam(t, r)
			h.backend.mu.Lock()
			tc.arrange(h, created.Id.ValueString())
			h.backend.mu.Unlock()

			readResp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
			if tc.wantDetail == "" {
				mustNoDiags(t, "Read", readResp.Diagnostics)
			} else if !hasDiag(readResp.Diagnostics, diag.SeverityError, tc.wantDetail) {
				t.Fatalf("diagnostics %+v, want an error containing %q", readResp.Diagnostics, tc.wantDetail)
			}
			if !readResp.State.Raw.Equal(state.Raw) {
				t.Fatal("Read changed the team's state")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestTeamDelete_LeavesTheTeam(t *testing.T) {
	cases := []struct {
		name        string
		arrange     func(h fakeHarness, id string)
		wantWarning string
		wantError   string
	}{
		// Fails if Delete raises no warning that the team stays.
		{name: "the token leaves and the team stays", wantWarning: "stay", arrange: func(fakeHarness, string) {}},
		// Fails if Delete ignores team_deleted.
		{name: "the token was the last member", wantWarning: "deleted the team", arrange: func(h fakeHarness, _ string) {
			h.backend.leaveDeletes = true
		}},
		// Fails if Delete treats NotFound as an error.
		{name: "already left", arrange: func(h fakeHarness, id string) {
			delete(h.backend.teams, id)
		}},
		// Fails if Delete swallows a refused leave.
		{name: "leave refused", wantError: "the last Admin cannot leave", arrange: func(h fakeHarness, _ string) {
			h.backend.leaveTeamErr = connect.NewError(connect.CodeFailedPrecondition, errors.New("the last Admin cannot leave"))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			h := newFakeClientSetT(t)
			r := &TeamResource{client: h.cs}
			state, created := startTeam(t, r)
			h.backend.mu.Lock()
			tc.arrange(h, created.Id.ValueString())
			h.backend.mu.Unlock()

			delResp := resource.DeleteResponse{}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &delResp)

			h.backend.mu.Lock()
			leaves := h.backend.leaveTeamReqs
			h.backend.mu.Unlock()
			if len(leaves) != 1 || leaves[0].TeamId != created.Id.ValueString() || leaves[0].DryRun {
				t.Fatalf("LeaveTeam calls: %+v, want one for %s", leaves, created.Id.ValueString())
			}
			if tc.wantError != "" {
				if !hasDiag(delResp.Diagnostics, diag.SeverityError, tc.wantError) {
					t.Fatalf("diagnostics %+v, want an error containing %q", delResp.Diagnostics, tc.wantError)
				}
				return
			}
			mustNoDiags(t, "Delete", delResp.Diagnostics)
			if tc.wantWarning == "" {
				if delResp.Diagnostics.WarningsCount() != 0 {
					t.Fatalf("warnings %+v, want none", delResp.Diagnostics)
				}
				return
			}
			if !hasDiag(delResp.Diagnostics, diag.SeverityWarning, tc.wantWarning) {
				t.Fatalf("diagnostics %+v, want a warning containing %q", delResp.Diagnostics, tc.wantWarning)
			}
		})
	}
}
