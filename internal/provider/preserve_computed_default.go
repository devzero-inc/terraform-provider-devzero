package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// The modifiers in this file exist because terraform-plugin-framework's
// schema.Default is unconditionally re-applied on every plan whenever an
// attribute is null in config (see the framework's internal
// fwschemadata.TransformDefaults) — it never checks whether a real, known
// value already exists in state. For any Optional+Computed attribute that
// also declares a Default, omitting the attribute from config resets it to
// the default on every single plan, clobbering the real backend value.
//
// This is invisible for resources this provider itself created (Create
// always re-applies the same default, so state and backend never diverge),
// but it silently destroys real data for any resource whose actual value
// differs from the static default — which includes every resource
// `terraform import` picks up that this exact Terraform config didn't
// create. Pairing Default with one of these modifiers restores the
// intended behavior: Default seeds a sane value at Create time (no prior
// state yet), and after that the real state value always wins over the
// default when config doesn't set the attribute explicitly.
const preserveStateOverDefaultDescription = "Once set (including by import), this attribute's real value is preserved across plans instead of reverting to its schema default when omitted from config."

func preserveStringStateOverDefault() planmodifier.String { return preserveStringOverDefault{} }
func preserveBoolStateOverDefault() planmodifier.Bool     { return preserveBoolOverDefault{} }
func preserveInt32StateOverDefault() planmodifier.Int32   { return preserveInt32OverDefault{} }
func preserveInt64StateOverDefault() planmodifier.Int64   { return preserveInt64OverDefault{} }
func preserveFloat32StateOverDefault() planmodifier.Float32 {
	return preserveFloat32OverDefault{}
}
func preserveListStateOverDefault() planmodifier.List { return preserveListOverDefault{} }

type preserveStringOverDefault struct{}

func (preserveStringOverDefault) Description(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveStringOverDefault) MarkdownDescription(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveStringOverDefault) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

type preserveBoolOverDefault struct{}

func (preserveBoolOverDefault) Description(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveBoolOverDefault) MarkdownDescription(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveBoolOverDefault) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

type preserveInt32OverDefault struct{}

func (preserveInt32OverDefault) Description(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveInt32OverDefault) MarkdownDescription(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveInt32OverDefault) PlanModifyInt32(_ context.Context, req planmodifier.Int32Request, resp *planmodifier.Int32Response) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

type preserveInt64OverDefault struct{}

func (preserveInt64OverDefault) Description(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveInt64OverDefault) MarkdownDescription(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveInt64OverDefault) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

type preserveFloat32OverDefault struct{}

func (preserveFloat32OverDefault) Description(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveFloat32OverDefault) MarkdownDescription(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveFloat32OverDefault) PlanModifyFloat32(_ context.Context, req planmodifier.Float32Request, resp *planmodifier.Float32Response) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

type preserveListOverDefault struct{}

func (preserveListOverDefault) Description(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveListOverDefault) MarkdownDescription(context.Context) string {
	return preserveStateOverDefaultDescription
}
func (preserveListOverDefault) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}
