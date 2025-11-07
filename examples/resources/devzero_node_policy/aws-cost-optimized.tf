# Cost-Optimized AWS Node Policy
#
# This example demonstrates a node policy optimized for cost savings:
# - Aggressive consolidation (5 minutes)
# - Spot instances prioritized
# - ARM architecture (Graviton) for better price/performance
# - Larger disruption budget (20%)
# - Modern instance generations only

resource "devzero_node_policy" "cost_optimized" {
  name            = "cost-optimized"
  description     = "Cost-optimized node policy for non-critical workloads"
  node_pool_name  = "cost-optimized-pool"
  node_class_name = "cost-optimized-class"
  weight          = 5 # Lower priority

  # Instance selection - modern, cost-effective instances
  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["c", "m"] # Compute and general purpose only
    }]
  }

  instance_generations = {
    match_expressions = [{
      key      = "instanceGenerations"
      operator = "Gt"
      values   = ["5"] # Latest generation for best pricing
    }]
  }

  # Use ARM architecture for better price/performance
  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["arm64"] # Graviton processors
    }]
  }

  # Prioritize spot instances for cost savings
  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["spot", "on-demand"] # Spot first, on-demand fallback
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  # Label nodes as spot-friendly
  labels = {
    "workload-type"     = "batch"
    "cost-optimization" = "aggressive"
    "interruption-ok"   = "true"
  }

  # Aggressive disruption policy for maximum cost savings
  disruption = {
    consolidate_after       = "5m" # Very aggressive consolidation
    consolidation_policy    = "WhenEmptyOrUnderutilized"
    expire_after            = "168h" # 7 days - rotate frequently for latest pricing
    ttl_seconds_after_empty = 30     # Terminate empty nodes quickly

    budgets = [
      {
        reasons = ["Underutilized", "Empty", "Drifted"]
        nodes   = "20%" # Allow more disruption for cost savings
      }
    ]
  }

  # Higher resource limits for batch workloads
  limits = {
    cpu    = "2000"
    memory = "4000Gi"
  }

  # AWS configuration optimized for cost
  aws = {
    role             = "KarpenterNodeRole-cost-optimized"
    instance_profile = "KarpenterNodeInstanceProfile-cost-optimized"
    ami_family       = "AL2023" # Amazon Linux 2023 is free

    ami_selector_terms = [
      {
        alias = "al2023@latest"
      }
    ]

    subnet_selector_terms = [
      {
        tags = {
          "karpenter.sh/discovery" = "production-cluster"
          "tier"                   = "private" # Private subnets for cost
        }
      }
    ]

    security_group_selector_terms = [
      {
        tags = {
          "karpenter.sh/discovery" = "production-cluster"
        }
      }
    ]

    tags = {
      "Environment"       = "production"
      "CostCenter"        = "batch-workloads"
      "OptimizationLevel" = "aggressive"
    }

    # Minimal block device for cost
    block_device_mappings = [
      {
        device_name = "/dev/xvda"
        ebs = {
          volume_size           = "50Gi" # Smaller volume
          volume_type           = "gp3"  # Cost-effective
          encrypted             = true
          delete_on_termination = true
        }
      }
    ]

    # Use instance store for ephemeral workloads
    instance_store_policy = "RAID0"

    # Disable detailed monitoring to save costs
    detailed_monitoring         = false
    associate_public_ip_address = false

    # Secure IMDS v2 (uses defaults)
    metadata_options = {
      http_endpoint = "enabled"
      http_tokens   = "required"
    }
  }
}

# Target this policy to batch workload clusters
resource "devzero_cluster" "batch_clusters" {
  name = "batch-processing-cluster"
}

resource "devzero_node_policy_target" "cost_optimized_target" {
  name        = "batch-workload-clusters"
  description = "Apply cost optimization to batch processing clusters"
  policy_id   = devzero_node_policy.cost_optimized.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.batch_clusters.id,
  ]
}
