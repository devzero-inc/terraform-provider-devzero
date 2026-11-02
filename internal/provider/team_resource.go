package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiv1 "github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"
)

var _ resource.Resource = &TeamResource{}
var _ resource.ResourceWithConfigure = &TeamResource{}
var _ resource.ResourceWithImportState = &TeamResource{}
var _ resource.ResourceWithModifyPlan = &TeamResource{}

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

// TeamResource starts a team from the provider's team, a DevZero partner team.
type TeamResource struct {
	client *ClientSet
}

type TeamResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	MemberEmails      types.Set    `tfsdk:"member_emails"`
	AdminInviteEmails types.Set    `tfsdk:"admin_invite_emails"`
	InvitesNotSent    types.Set    `tfsdk:"invites_not_sent"`
}

const teamResourceDescription = "A team started from the provider's team, such as a partner's team for one of its customers. " +
	"Creating it starts the team in DevZero with the provider's `team_id` as the starting team, which must be a DevZero partner team.\n\n" +
	"Drive this resource with a service account's token (`dzs-`), not a person's. Destroying it is the token owner's leave of the team: " +
	"with a service account's token the team, its people and its clusters stay. With a person's token the destroy is that person's leave, " +
	"and when they are the team's last member it deletes the team.\n\n" +
	"The values the team is started with are not managed afterwards: the team's Admins rename it and manage its people in DevZero, " +
	"and a changed `name`, `member_emails` or `admin_invite_emails` is refused. " +
	"If the team can no longer be read (it was deleted, or the token's owner was removed from it), refresh fails rather than starting a new team."

func (r *TeamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: teamResourceDescription,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the team",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the team when it is started. A rename by the team's Admins shows no drift; a change here is refused.",
				Required:            true,
			},
			"member_emails": schema.SetAttribute{
				MarkdownDescription: "People of the provider's team, by email, who join the new team, each in the role they hold in the provider's team (Admin or Developer). " +
					"At least one person must join as Admin, so with a service account's token name an Admin here. " +
					"Used only when the team is started: a change is refused; manage the team's people in its Settings > Members.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"admin_invite_emails": schema.SetAttribute{
				MarkdownDescription: "Addresses invited as Admin of the new team when it is started, such as the customer's admin. " +
					"A person already in the provider's team is named in `member_emails` instead. " +
					"Used only when the team is started: a change is refused; send further invites from the team's Settings > Members.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"invites_not_sent": schema.SetAttribute{
				MarkdownDescription: "Addresses from `admin_invite_emails` whose invite was saved but whose email DevZero could not send. Re-send them from the team's Settings > Members.",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientSet)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ClientSet, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// ModifyPlan refuses a change to the values a team was started with, since
// nothing sends them after the start. An imported team has no name in state:
// its plan takes the configured values, and Update makes no call.
func (r *TeamResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var prior, planned TeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	if resp.Diagnostics.HasError() || prior.Name.IsNull() {
		return
	}

	refuse := func(attribute string, was, now attr.Value, where string) {
		if now.IsUnknown() || now.Equal(was) {
			return
		}
		resp.Diagnostics.AddAttributeError(
			path.Root(attribute),
			fmt.Sprintf("%s is used only when the team is started", attribute),
			fmt.Sprintf("Team %s was started with %s = %s. Terraform does not change it afterwards. %s Then set %s back to its value at the start.",
				prior.Id.ValueString(), attribute, was.String(), where, attribute),
		)
	}
	refuse("name", prior.Name, planned.Name,
		"The team's Admins rename it in DevZero; a Developer, as a service account is, cannot.")
	refuse("member_emails", prior.MemberEmails, planned.MemberEmails,
		"The team's Admins add and remove people in its Settings > Members.")
	refuse("admin_invite_emails", prior.AdminInviteEmails, planned.AdminInviteEmails,
		"The team's Admins send invites from its Settings > Members.")
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	memberEmails := emailsFromSet(ctx, data.MemberEmails, &resp.Diagnostics)
	adminInviteEmails := emailsFromSet(ctx, data.AdminInviteEmails, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.TeamClient.CreateTeam(ctx, connect.NewRequest(&apiv1.CreateTeamRequest{
		TeamId:            r.client.TeamId,
		Name:              data.Name.ValueString(),
		MemberEmails:      memberEmails,
		AdminInviteEmails: adminInviteEmails,
	}))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to start team",
			fmt.Sprintf("DevZero refused to start team %q from team %s: %s", data.Name.ValueString(), r.client.TeamId, err),
		)
		return
	}
	if createResp.Msg.Team == nil || createResp.Msg.Team.Id == "" {
		resp.Diagnostics.AddError("Client Error", "Team not created")
		return
	}

	notSent := append([]string(nil), createResp.Msg.InvitesNotSent...)
	sort.Strings(notSent)
	elems := make([]attr.Value, 0, len(notSent))
	for _, email := range notSent {
		elems = append(elems, types.StringValue(email))
	}

	data.Id = types.StringValue(createResp.Msg.Team.Id)
	data.InvitesNotSent = types.SetValueMust(types.StringType, elems)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if len(notSent) > 0 {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("invites_not_sent"),
			"Invite emails not sent",
			fmt.Sprintf("Team %s was started and its invites were saved, but DevZero could not email %s. Re-send them from the team's Settings > Members.",
				createResp.Msg.Team.Id, strings.Join(notSent, ", ")),
		)
	}
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The state keeps the values the team was started with, so nothing from
	// the response is written back.
	getResp, err := r.client.TeamClient.GetTeam(ctx, connect.NewRequest(&apiv1.GetTeamRequest{TeamId: data.Id.ValueString()}))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team %s, got error: %s", data.Id.ValueString(), err))
		return
	}
	if err != nil || getResp.Msg.Team == nil {
		resp.Diagnostics.AddError(
			"Team not found",
			fmt.Sprintf("Team %s is gone, or this provider's token is no longer in it. Terraform keeps it in state rather than starting a new team. "+
				"Remove this devzero_team and the devzero_cluster resources in its team from configuration as well as from state "+
				"(terraform state rm), or replace them with removed blocks. "+
				"Removed from state but kept in configuration, they start a new team and register the customer's clusters into it on the next apply.",
				data.Id.ValueString()),
		)
		return
	}
}

// Update only records configured values in state: ModifyPlan lets a change
// through only after an import, and nothing is sent for it.
func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planned, prior TeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned.Id = prior.Id
	if planned.InvitesNotSent.IsUnknown() {
		planned.InvitesNotSent = prior.InvitesNotSent
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &planned)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	leaveResp, err := r.client.TeamClient.LeaveTeam(ctx, connect.NewRequest(&apiv1.LeaveTeamRequest{TeamId: data.Id.ValueString()}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return
		}
		resp.Diagnostics.AddError("Unable to leave team", fmt.Sprintf("Unable to leave team %s, got error: %s", data.Id.ValueString(), err))
		return
	}

	if leaveResp.Msg.TeamDeleted {
		resp.Diagnostics.AddWarning(
			"Team deleted",
			fmt.Sprintf("This provider's token was the last member of team %s, so leaving it deleted the team.", data.Id.ValueString()),
		)
		return
	}
	resp.Diagnostics.AddWarning(
		"Left the team; the team stays",
		fmt.Sprintf("This provider's token left team %s. The team, its people and its clusters stay in DevZero, where its Admins manage or delete them.", data.Id.ValueString()),
	)
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func emailsFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var emails []string
	diags.Append(set.ElementsAs(ctx, &emails, false)...)
	sort.Strings(emails)
	return emails
}
