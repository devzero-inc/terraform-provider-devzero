# Complete Azure Node Policy Example
#
# This example demonstrates a comprehensive Azure node policy with:
# - Azure-specific configuration
# - Instance selection for Azure VM sizes
# - Cost optimization with spot instances
# - Proper VNET and subnet configuration

resource "devzero_node_policy" "azure_production" {
  name            = "azure-production"
  description     = "Production node policy for Azure AKS clusters"
  node_pool_name  = "azure-production-pool"
  node_class_name = "azure-production-class"
  weight          = 15

  # Azure instance categories (D, E, F series)
  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["D", "E"] # General purpose and memory optimized
    }]
  }

  # Modern instance generations
  instance_generations = {
    match_expressions = [{
      key      = "instanceGenerations"
      operator = "Gt"
      values   = ["4"] # v5 series and newer
    }]
  }

  # Architecture
  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["amd64"]
    }]
  }

  # Capacity types - use spot for cost savings
  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["spot", "on-demand"]
    }]
  }

  # Operating system
  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  # Labels for node identification
  labels = {
    "dedicated"     = "karpenter"
    "cloud"         = "azure"
    "workload-type" = "general"
  }

  # Taints for workload isolation
  taints = [
    {
      key    = "karpenter"
      value  = "true"
      effect = "NoSchedule"
    }
  ]

  # Aggressive disruption policy for Azure (as per your example)
  disruption = {
    consolidate_after    = "5m" # Aggressive consolidation
    consolidation_policy = "WhenEmptyOrUnderutilized"
    expire_after         = "168h" # 7 days

    budgets = [
      {
        reasons = ["Empty", "Drifted", "Underutilized"]
        nodes   = "20%" # More aggressive than AWS
      }
    ]
  }

  # Resource limits
  limits = {
    cpu    = "500"
    memory = "1000Gi"
  }

  # Azure-specific configuration
  azure = {
    # VNET subnet - must be in same region as cluster
    vnet_subnet_id = "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/production-rg/providers/Microsoft.Network/virtualNetworks/production-vnet/subnets/karpenter-subnet"

    # OS disk configuration
    os_disk_size_gb = 128 # Adequate for most workloads

    # Image family
    image_family = "Ubuntu2204" # Ubuntu 22.04 LTS

    # FIPS mode (required for compliance in some industries)
    fips_mode = "Disabled" # Set to "FIPS" if required

    # Maximum pods per node
    max_pods = 110 # Azure CNI default

    # Azure tags
    tags = {
      "Environment" = "production"
      "ManagedBy"   = "Karpenter"
      "Team"        = "platform"
      "CostCenter"  = "engineering"
      "Project"     = "platform-infrastructure"
    }
  }
}

# Example with different Azure regions and requirements
resource "devzero_node_policy" "azure_gpu" {
  name            = "azure-gpu-workloads"
  description     = "GPU-enabled node policy for ML workloads in Azure"
  node_pool_name  = "gpu-pool"
  node_class_name = "gpu-class"
  weight          = 25 # Higher priority for GPU workloads

  # Select GPU instance families
  instance_families = {
    match_expressions = [{
      key      = "instanceFamilies"
      operator = "In"
      values   = ["NC", "ND", "NV"] # GPU families
    }]
  }

  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["amd64"]
    }]
  }

  # On-demand only for GPU (spot availability is limited)
  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["on-demand"]
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  # GPU-specific labels
  labels = {
    "workload-type" = "gpu"
    "accelerator"   = "nvidia"
    "ml-ready"      = "true"
  }

  # GPU workload taints
  taints = [
    {
      key    = "nvidia.com/gpu"
      value  = "true"
      effect = "NoSchedule"
    }
  ]

  # Conservative disruption for expensive GPU nodes
  disruption = {
    consolidate_after       = "30m"       # Wait longer before consolidating
    consolidation_policy    = "WhenEmpty" # Only consolidate when completely empty
    expire_after            = "720h"      # 30 days
    ttl_seconds_after_empty = 900         # 15 minutes

    budgets = [
      {
        reasons = ["Empty"]
        nodes   = "5%" # Very conservative for expensive nodes
      }
    ]
  }

  # Higher limits for GPU workloads
  limits = {
    cpu    = "200"
    memory = "500Gi"
  }

  azure = {
    vnet_subnet_id  = "/subscriptions/12345678-1234-1234-1234-123456789abc/resourceGroups/ml-rg/providers/Microsoft.Network/virtualNetworks/ml-vnet/subnets/gpu-subnet"
    os_disk_size_gb = 256 # Larger disk for ML models and data
    image_family    = "Ubuntu2204"
    fips_mode       = "Disabled"
    max_pods        = 30 # Fewer pods for GPU workloads

    tags = {
      "Environment" = "production"
      "ManagedBy"   = "Karpenter"
      "Team"        = "ml-engineering"
      "CostCenter"  = "ml-research"
      "GPU"         = "enabled"
    }
  }
}

# Complete example with targets
resource "devzero_cluster" "azure_us_east" {
  name = "aks-us-east-prod"
}

resource "devzero_cluster" "azure_eu_west" {
  name = "aks-eu-west-prod"
}

resource "devzero_node_policy_target" "azure_production_target" {
  name        = "azure-production-clusters"
  description = "Apply production node policy to all Azure clusters"
  policy_id   = devzero_node_policy.azure_production.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.azure_us_east.id,
    devzero_cluster.azure_eu_west.id,
  ]
}

resource "devzero_cluster" "azure_ml_cluster" {
  name = "aks-ml-prod"
}

resource "devzero_node_policy_target" "azure_gpu_target" {
  name        = "azure-gpu-ml-cluster"
  description = "Apply GPU node policy to ML cluster"
  policy_id   = devzero_node_policy.azure_gpu.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.azure_ml_cluster.id,
  ]
}
