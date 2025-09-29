resource "devzero_cluster" "cluster" {
  name = "terraform-example"
}

resource "devzero_workload_policy" "workload_policy" {
  name = "terraform-example"
}

# Only required attributes
resource "devzero_workload_policy_target" "workload_policy_target" {
  name        = "terraform-example"
  policy_id   = devzero_workload_policy.workload_policy.id
  cluster_ids = [devzero_cluster.cluster.id]
}

# All attributes
resource "devzero_workload_policy_target" "workload_policy_target" {
  name        = "terraform-example"
  description = "some description"
  policy_id   = devzero_workload_policy.workload_policy.id
  cluster_ids = [devzero_cluster.cluster.id]
  priority    = 1
  enabled     = true

  workload_names   = ["workload-1", "workload-2"]     # Empty list means all workloads
  node_group_names = ["node-group-1", "node-group-2"] # Empty list means all node groups
  kind_filter      = ["Deployment", "ReplicaSet"]     # Empty list means all kinds

  name_pattern = {
    pattern = "terraform-example"
    flags   = "i"
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

  annotation_selector = {
    match_labels = {
      app = "terraform-example"
    }
  }
}