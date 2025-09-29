# Only required attributes
resource "devzero_workload_policy" "workload_policy" {
  name            = "terraform-example"
  action_triggers = ["on_detection", "on_schedule"]
}

# All attributes
resource "devzero_workload_policy" "workload_policy" {
  name                     = "terraform-example"
  description              = "some description"
  action_triggers          = ["on_detection", "on_schedule"]
  cron_schedule            = "*/15 * * * *" # Every 15th minute
  detection_triggers       = ["pod_creation", "pod_update"]
  recommendation_mode      = "balanced"
  loopback_period_seconds  = 3600 # 1 hour
  startup_period_seconds   = 60 # 1 minute
  live_migration_enabled   = true
  scheduler_plugins        = ["dz-scheduler"]
  defragmentation_schedule = "*/15 * * * *"

  cpu_vertical_scaling = {
    enabled                   = true
    min_request               = 1000
    max_request               = 2000
    overhead_multiplier       = 0.05 # 5%
    limits_adjustment_enabled = true
  }

  memory_vertical_scaling = {
    enabled                   = true
    min_request               = 1000
    max_request               = 2000
    overhead_multiplier       = 0.05 # 5%
    limits_adjustment_enabled = true
  }

  gpu_vertical_scaling = {
    enabled                   = true
    min_request               = 1000
    max_request               = 2000
    overhead_multiplier       = 0.05 # 5%
    limits_adjustment_enabled = false
  }

  gpu_vram_vertical_scaling = {
    enabled                   = true
    min_request               = 1000
    max_request               = 2000
    overhead_multiplier       = 0.05 # 5%
    limits_adjustment_enabled = false
  }

  horizontal_scaling = {
    enabled                   = true
    min_replicas              = 1
    max_replicas              = 2
    overhead_multiplier       = 0.05 # 5%
    limits_adjustment_enabled = true
  }
}