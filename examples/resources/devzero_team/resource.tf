# A partner onboards a customer: it starts the customer's team from its own
# team and registers the customer's cluster in it. The provider runs as the
# partner's team, with a service account's token kept in the partner's
# secret store.
variable "partner_team_id" {
  type = string
}

variable "partner_service_account_token" {
  type      = string
  sensitive = true
}

provider "devzero" {
  url     = "https://dakr.devzero.io"
  team_id = var.partner_team_id
  token   = var.partner_service_account_token
}

# The people named in member_emails join in the role they hold in the
# partner's team, and one of them must be an Admin there. The customer's
# admin is invited as Admin and joins when they accept the invite.
resource "devzero_team" "acme" {
  name                = "Acme"
  member_emails       = ["alice@partner.example"]
  admin_invite_emails = ["admin@acme.example"]
}

# Register the customer's cluster in the new team. Its token is the one
# DevZero's agents in that cluster report with.
resource "devzero_cluster" "acme_prod" {
  name    = "acme-prod"
  team_id = devzero_team.acme.id
}
