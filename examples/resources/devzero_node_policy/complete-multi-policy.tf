# Complete Multi-Policy Example
#
# This example demonstrates a real-world setup with multiple node policies
# for different workload types, each applied to specific clusters.
#
# Architecture:
# - Production workloads: Stable, on-demand nodes
# - Development workloads: Cost-optimized, spot nodes
# - Batch workloads: Aggressive cost optimization
# - GPU workloads: Specialized, expensive instances

# ============================================================================
# Clusters
# ============================================================================

resource "devzero_cluster" "prod_us_east" {
  name = "production-us-east-1"
}

resource "devzero_cluster" "prod_us_west" {
  name = "production-us-west-2"
}

resource "devzero_cluster" "dev_us_east" {
  name = "development-us-east-1"
}

resource "devzero_cluster" "batch_us_west" {
  name = "batch-processing-us-west-2"
}

# ============================================================================
# Production Policy - Stability First
# ============================================================================

resource "devzero_node_policy" "production" {
  name            = "production-stable"
  description     = "Stable configuration for production workloads"
  node_pool_name  = "production-pool"
  node_class_name = "production-class"
  weight          = 20 # High priority

  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["c", "m", "r"]
    }]
  }

  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["amd64"]
    }]
  }

  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["on-demand", "spot"] # On-demand preferred
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  labels = {
    "environment" = "production"
    "stability"   = "high"
  }

  disruption = {
    consolidate_after    = "2h"
    consolidation_policy = "WhenEmptyOrUnderutilized"
    expire_after         = "720h" # 30 days

    budgets = [{
      reasons = ["Underutilized", "Empty"]
      nodes   = "10%"
    }]
  }

  limits = {
    cpu    = "1000"
    memory = "2000Gi"
  }

  aws = {
    role       = "KarpenterNodeRole-production"
    ami_family = "AL2023"

    ami_selector_terms = [{
      alias = "al2023@latest"
    }]

    subnet_selector_terms = [{
      tags = {
        "karpenter.sh/discovery" = "production-cluster"
      }
    }]

    security_group_selector_terms = [{
      tags = {
        "karpenter.sh/discovery" = "production-cluster"
      }
    }]

    tags = {
      "Environment" = "production"
      "ManagedBy"   = "Karpenter"
    }

    block_device_mappings = [{
      device_name = "/dev/xvda"
      ebs = {
        volume_size           = "200Gi"
        volume_type           = "gp3"
        encrypted             = true
        delete_on_termination = true
      }
    }]

    detailed_monitoring = true
  }
}

resource "devzero_node_policy_target" "production" {
  name        = "production-clusters"
  description = "Production clusters in all regions"
  policy_id   = devzero_node_policy.production.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.prod_us_east.id,
    devzero_cluster.prod_us_west.id,
  ]
}

# ============================================================================
# Development Policy - Cost Optimized
# ============================================================================

resource "devzero_node_policy" "development" {
  name            = "development-cost-optimized"
  description     = "Cost-optimized configuration for development workloads"
  node_pool_name  = "development-pool"
  node_class_name = "development-class"
  weight          = 10 # Medium priority

  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["c", "m"]
    }]
  }

  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["arm64"] # Graviton for cost savings
    }]
  }

  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["spot", "on-demand"] # Spot preferred
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  labels = {
    "environment"       = "development"
    "cost-optimization" = "enabled"
  }

  disruption = {
    consolidate_after       = "10m"
    consolidation_policy    = "WhenEmptyOrUnderutilized"
    expire_after            = "168h" # 7 days
    ttl_seconds_after_empty = 60

    budgets = [{
      reasons = ["Underutilized", "Empty", "Drifted"]
      nodes   = "30%"
    }]
  }

  limits = {
    cpu    = "500"
    memory = "1000Gi"
  }

  aws = {
    role       = "KarpenterNodeRole-development"
    ami_family = "AL2023"

    ami_selector_terms = [{
      alias = "al2023@latest"
    }]

    subnet_selector_terms = [{
      tags = {
        "karpenter.sh/discovery" = "development-cluster"
      }
    }]

    security_group_selector_terms = [{
      tags = {
        "karpenter.sh/discovery" = "development-cluster"
      }
    }]

    tags = {
      "Environment" = "development"
      "ManagedBy"   = "Karpenter"
      "CostCenter"  = "development"
    }

    block_device_mappings = [{
      device_name = "/dev/xvda"
      ebs = {
        volume_size           = "50Gi"
        volume_type           = "gp3"
        encrypted             = true
        delete_on_termination = true
      }
    }]

    detailed_monitoring = false # Save costs
  }
}

resource "devzero_node_policy_target" "development" {
  name        = "development-clusters"
  description = "Development clusters"
  policy_id   = devzero_node_policy.development.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.dev_us_east.id,
  ]
}

# ============================================================================
# Batch Processing Policy - Maximum Cost Optimization
# ============================================================================

resource "devzero_node_policy" "batch" {
  name            = "batch-aggressive-cost"
  description     = "Aggressive cost optimization for batch workloads"
  node_pool_name  = "batch-pool"
  node_class_name = "batch-class"
  weight          = 5 # Low priority

  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["c"] # Compute-optimized only
    }]
  }

  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["arm64"]
    }]
  }

  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["spot"] # Spot only!
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  labels = {
    "workload-type"     = "batch"
    "interruption-ok"   = "true"
    "cost-optimization" = "aggressive"
  }

  taints = [{
    key    = "batch-workload"
    value  = "true"
    effect = "NoSchedule"
  }]

  disruption = {
    consolidate_after       = "3m" # Very aggressive
    consolidation_policy    = "WhenEmptyOrUnderutilized"
    expire_after            = "24h" # Short-lived
    ttl_seconds_after_empty = 30

    budgets = [{
      reasons = ["Underutilized", "Empty", "Drifted"]
      nodes   = "50%" # Very aggressive
    }]
  }

  limits = {
    cpu    = "2000"
    memory = "4000Gi"
  }

  aws = {
    role       = "KarpenterNodeRole-batch"
    ami_family = "AL2023"

    ami_selector_terms = [{
      alias = "al2023@latest"
    }]

    subnet_selector_terms = [{
      tags = {
        "karpenter.sh/discovery" = "batch-cluster"
      }
    }]

    security_group_selector_terms = [{
      tags = {
        "karpenter.sh/discovery" = "batch-cluster"
      }
    }]

    tags = {
      "Environment" = "batch"
      "ManagedBy"   = "Karpenter"
      "CostCenter"  = "batch-processing"
      "Spot"        = "only"
    }

    block_device_mappings = [{
      device_name = "/dev/xvda"
      ebs = {
        volume_size           = "30Gi"
        volume_type           = "gp3"
        encrypted             = true
        delete_on_termination = true
      }
    }]

    instance_store_policy = "RAID0"
    detailed_monitoring   = false
  }
}

resource "devzero_node_policy_target" "batch" {
  name        = "batch-processing-clusters"
  description = "Batch processing clusters"
  policy_id   = devzero_node_policy.batch.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.batch_us_west.id,
  ]
}

# ============================================================================
# Outputs
# ============================================================================

output "production_policy_id" {
  description = "ID of the production node policy"
  value       = devzero_node_policy.production.id
}

output "development_policy_id" {
  description = "ID of the development node policy"
  value       = devzero_node_policy.development.id
}

output "batch_policy_id" {
  description = "ID of the batch node policy"
  value       = devzero_node_policy.batch.id
}

output "all_targets" {
  description = "All node policy targets"
  value = {
    production = {
      id          = devzero_node_policy_target.production.id
      cluster_ids = devzero_node_policy_target.production.cluster_ids
    }
    development = {
      id          = devzero_node_policy_target.development.id
      cluster_ids = devzero_node_policy_target.development.cluster_ids
    }
    batch = {
      id          = devzero_node_policy_target.batch.id
      cluster_ids = devzero_node_policy_target.batch.cluster_ids
    }
  }
}
