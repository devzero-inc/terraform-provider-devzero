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

  action_triggers    = ["on_schedule", "on_detection"]
  cron_schedule      = "0 2 * * *"
  detection_triggers = ["pod_creation", "pod_update", "pod_evict"]

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
    enabled      = true
    min_replicas = 1
    max_replicas = 10

    # Metric triggers: built-in CPU/Memory or external (e.g. Prometheus)
    metrics = [
      {
        type               = "CPU"
        target_utilization = "0.70"
      },
      {
        type           = "prometheus"
        target_value   = "100"
        server_address = "http://prometheus.monitoring.svc.cluster.local:9090"
        query          = "rate(http_requests_total{job=\"my-api\"}[5m])"
        metadata = {
          "customKey" = "customValue"
        }
      }
    ]
  }

  emergency_response = {
    oom_enabled               = true
    oom_memory_multiplier     = 1.5
    cpu_throttling_enabled    = true
    cpu_throttling_threshold  = 0.20
    cpu_throttling_multiplier = 1.25
  }

  # JVM heap sizing (only applies when the workload is detected as running a JVM)
  jvm_heap_rule = {
    enabled                   = true
    target_percentile         = 0.95
    headroom_multiplier       = 1.2
    non_heap_overhead_percent = 0.15
    min_heap_bytes            = 268435456  # 256Mi
    max_heap_bytes            = 4294967296 # 4Gi
    prefer_container_support  = false
  }
  jvm_cpu_startup_floor_millicores = 250 # override the 75m default while the JVM warms up

  # Hand the ScaledObject lifecycle to KEDA instead of generating an HPA
  keda_scaled_object = {
    min_replica_count = 1
    max_replica_count = 20
    cooldown_period   = 300

    triggers = [
      {
        type = "prometheus"
        metadata = {
          serverAddress = "http://prometheus.monitoring.svc.cluster.local:9090"
          query         = "rate(http_requests_total{job=\"my-api\"}[5m])"
          threshold     = "100"
        }
      }
    ]

    fallback = {
      failure_threshold = 3
      replicas          = 2
    }
  }

  live_migration_enabled               = false
  use_in_place_vertical_scaling        = false
  allow_in_place_memory_limit_decrease = false
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
