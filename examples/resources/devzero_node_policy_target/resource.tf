# Prerequisites - need cluster and node policy resources
resource "devzero_cluster" "production" {
  name = "production-cluster"
}

resource "devzero_node_policy" "general" {
  name            = "general-purpose"
  node_pool_name  = "general-pool"
  node_class_name = "general-class"
}

# Minimal example - only required attributes
resource "devzero_node_policy_target" "minimal" {
  name        = "production-clusters"
  policy_id   = devzero_node_policy.general.id
  cluster_ids = [devzero_cluster.production.id]

  # Defaults applied automatically:
  # - description = ""
  # - enabled = true
}

# Comprehensive example - all attributes
resource "devzero_node_policy_target" "comprehensive" {
  name        = "production-general-target"
  description = "Applies general purpose node policy to production clusters"
  policy_id   = devzero_node_policy.general.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.production.id,
    # Add more cluster IDs as needed
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
  policy_id   = devzero_node_policy.general.id
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
  policy_id   = devzero_node_policy.general.id
  enabled     = false # Target exists but is not active
  cluster_ids = [devzero_cluster.production.id]
}
