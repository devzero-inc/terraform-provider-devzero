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

# All attributes
resource "devzero_workload_policy_target" "full" {
  name        = "terraform-example"
  description = "some description"
  policy_id   = devzero_workload_policy.cost_saving.id
  cluster_ids = [devzero_cluster.production.id]
  priority    = 1
  enabled     = true

  workload_names   = ["workload-1", "workload-2"]     # Empty list means all workloads
  node_group_names = ["node-group-1", "node-group-2"] # Empty list means all node groups
  kind_filter      = ["Deployment", "StatefulSet"]    # Empty list means all kinds

  name_pattern = {
    pattern = "terraform-example"
    flags   = "i"
  }

  namespace_pattern = {
    pattern = "^prod-"
    flags   = "i" # case-insensitive
  }

  namespace_selector = {
    match_labels = {
      app = "terraform-example"
    }
  }

  workload_selector = {
    match_labels = {
      app = "terraform-example"
    }
  }
}
