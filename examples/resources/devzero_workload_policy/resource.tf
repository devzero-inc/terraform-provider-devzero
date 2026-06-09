# Minimal — only required attributes
resource "devzero_workload_policy" "minimal" {
  name            = "cost-saving-policy"
  action_triggers = ["on_detection", "on_schedule"]
}

# Full example — values kept in sync with the Pulumi provider
resource "devzero_workload_policy" "cost_saving" {
  name                    = "cost-saving-policy"
  description             = "Rightsize non-critical workloads"
  action_triggers         = ["on_detection", "on_schedule"]
  cron_schedule           = "*/15 * * * *"                 # every 15 min; required when "on_schedule" is set
  detection_triggers      = ["pod_creation", "pod_update"] # used when "on_detection" is set
  loopback_period_seconds = 86400                          # 1 day — lookback window for usage data
  cooldown_minutes        = 300                            # 5 hours between successive scale-down actions
  min_data_points         = 20                             # min samples before any recommendation
  min_change_percent      = 0.2                            # apply only if change > 20%

  cpu_vertical_scaling = {
    enabled                    = true
    target_percentile          = 0.75 # P75 of observed usage
    min_request                = 25   # millicores; hard floor
    max_scale_up_percent       = 1000 # max % increase per step
    max_scale_down_percent     = 1    # max % decrease per step
    min_data_points            = 20   # min CPU samples before recommendation
    adjust_req_even_if_not_set = true # set requests even if workload has none
    limits_removal_enabled     = true # strip CPU limits (cycles compress safely)
  }

  memory_vertical_scaling = {
    enabled                    = true
    target_percentile          = 1         # P100 — guard against OOMKills
    min_request                = 134217728 # 128 MiB in bytes; hard floor
    max_scale_up_percent       = 1000      # max % increase per step
    max_scale_down_percent     = 1         # max % decrease per step
    overhead_multiplier        = 0.3       # extra headroom over the recommendation
    limits_adjustment_enabled  = true      # adjust limits alongside requests
    limit_multiplier           = 1         # limits = request × this
    min_data_points            = 20        # min memory samples before recommendation
    adjust_req_even_if_not_set = true      # set requests even if workload has none
    limits_removal_enabled     = false     # memory limits removal not supported
  }

  enable_pmax_protection = true # guard against spike-induced OOMKills
  pmax_ratio_threshold   = 3    # raise requests when peak is 3× the recommendation
}