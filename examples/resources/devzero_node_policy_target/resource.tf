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

# Example with multiple clusters
resource "devzero_cluster" "us_east" {
  name = "production-us-east-1"
}

resource "devzero_cluster" "us_west" {
  name = "production-us-west-2"
}

resource "devzero_cluster" "eu_west" {
  name = "production-eu-west-1"
}

resource "devzero_node_policy_target" "multi_cluster" {
  name        = "all-production-clusters"
  description = "Apply cost optimization policy to all production clusters"
  policy_id   = devzero_node_policy.standard_nodes.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.us_east.id,
    devzero_cluster.us_west.id,
    devzero_cluster.eu_west.id,
  ]
}

# Example of disabled target (for temporary disabling without destroying)
resource "devzero_node_policy_target" "disabled" {
  name        = "staging-clusters"
  description = "Temporarily disabled while testing new policy"
  policy_id   = devzero_node_policy.standard_nodes.id
  enabled     = false # Target exists but is not active
  cluster_ids = [devzero_cluster.production.id]
}
