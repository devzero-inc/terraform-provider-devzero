resource "devzero_cluster" "production" {
  name = "production-cluster"
}

resource "devzero_workload_policy" "cost_saving" {
  name = "cost-saving-policy"
}

# Minimal — only required attributes
resource "devzero_workload_policy_target" "minimal" {
  name        = "production-target"
  policy_id   = devzero_workload_policy.cost_saving.id
  cluster_ids = [devzero_cluster.production.id]
}

# Full example — values kept in sync with the Pulumi provider
resource "devzero_workload_policy_target" "production" {
  name        = "production-target"
  policy_id   = devzero_workload_policy.cost_saving.id
  cluster_ids = [devzero_cluster.production.id]
  kind_filter = ["Deployment", "StatefulSet"]
  enabled     = true

  # Match namespaces by name pattern — useful when namespaces follow a naming
  # convention but aren't consistently labeled (e.g. team-*, prod-*).
  namespace_pattern = {
    pattern = "^prod-"
    flags   = "i" # case-insensitive
  }
}