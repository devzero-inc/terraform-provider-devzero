# Minimal — auto-generate all fields
resource "devzero_workload_rule" "auto" {
  cluster_id    = "<YOUR_CLUSTER_ID>"
  namespace     = "production"
  kind          = "Deployment"
  name          = "my-api"
  auto_generate = true
}

# Manual — full control over scaling rules
resource "devzero_workload_rule" "manual" {
  cluster_id = "<YOUR_CLUSTER_ID>"
  namespace  = "production"
  kind       = "Deployment"
  name       = "my-api"

  action_triggers    = ["on_schedule"]
  cron_schedule      = "0 2 * * *"
  detection_triggers = ["pod_creation", "pod_update"]

  cpu_rule = {
    enabled                   = true
    min_request               = 10
    max_request               = 32000
    target_percentile         = 0.95
    limits_adjustment_enabled = true
    limit_multiplier          = 1.0
  }

  memory_rule = {
    enabled                   = true
    min_request               = 67108864    # 64Mi in bytes
    max_request               = 68719476736 # 64Gi in bytes
    target_percentile         = 0.95
    limits_adjustment_enabled = true
  }

  hpa_rule = {
    enabled            = true
    min_replicas       = 1
    max_replicas       = 10
    target_utilization = 0.70
    primary_metric     = "cpu"
  }

  emergency_response = {
    oom_enabled               = true
    oom_memory_multiplier     = 1.5
    cpu_throttling_enabled    = true
    cpu_throttling_threshold  = 0.20
    cpu_throttling_multiplier = 1.25
  }

  live_migration_enabled        = false
  use_in_place_vertical_scaling = false
}

# Per-container rules
resource "devzero_workload_rule" "per_container" {
  cluster_id = "<YOUR_CLUSTER_ID>"
  namespace  = "production"
  kind       = "Deployment"
  name       = "my-multi-container-app"

  action_triggers    = ["on_detection"]
  detection_triggers = ["pod_creation"]

  containers = [
    {
      container_name = "app"
      cpu_rule = {
        enabled     = true
        min_request = 10
        max_request = 32000
      }
      memory_rule = {
        enabled     = true
        min_request = 67108864    # 64Mi
        max_request = 68719476736 # 64Gi
      }
    },
    {
      container_name = "sidecar"
      cpu_rule = {
        enabled     = true
        max_request = 500
      }
    }
  ]
}
