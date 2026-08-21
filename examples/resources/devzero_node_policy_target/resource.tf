# Prerequisites
resource "devzero_cluster" "production" {
  name = "production-cluster"
}

resource "devzero_node_policy" "standard_nodes" {
  name = "standard-nodes"
}

# Minimal example - only required attributes
resource "devzero_node_policy_target" "minimal" {
  name        = "production-clusters"
  policy_id   = devzero_node_policy.standard_nodes.id
  cluster_ids = [devzero_cluster.production.id]

  # Defaults applied automatically:
  # - description = ""
  # - enabled = true
}

# Comprehensive example - all attributes
resource "devzero_node_policy_target" "comprehensive" {
  name        = "cluster-nodes"
  description = "Applies standard node policy to production clusters"
  policy_id   = devzero_node_policy.standard_nodes.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.production.id,
  ]
}

# The API allows at most ONE cluster per target — to cover several clusters,
# create one target per cluster (for_each keeps it concise).
resource "devzero_cluster" "production_regions" {
  for_each = toset(["us-east-1", "us-west-2", "eu-west-1"])
  name     = "production-${each.key}"
}

resource "devzero_node_policy_target" "per_cluster" {
  for_each    = devzero_cluster.production_regions
  name        = "standard-nodes-${each.key}"
  description = "Apply cost optimization policy to ${each.key}"
  policy_id   = devzero_node_policy.standard_nodes.id
  enabled     = true
  cluster_ids = [each.value.id]
}

# Example of disabled target (for temporary disabling without destroying)
resource "devzero_node_policy_target" "disabled" {
  name        = "staging-clusters"
  description = "Temporarily disabled while testing new policy"
  policy_id   = devzero_node_policy.standard_nodes.id
  enabled     = false # Target exists but is not active
  cluster_ids = [devzero_cluster.production.id]
}
