# Minimal example - uses sensible defaults
resource "devzero_node_policy" "minimal" {
  name            = "minimal-policy"
  node_pool_name  = "default-pool"
  node_class_name = "default-class"

  # Defaults applied automatically:
  # - weight = 10 (medium priority)
  # - disruption.consolidate_after = "15m"
  # - disruption.consolidation_policy = "WhenEmptyOrUnderutilized"
  # - disruption.expire_after = "720h"
}

# AWS example with required fields only
resource "devzero_node_policy" "aws_basic" {
  name            = "aws-basic-policy"
  node_pool_name  = "production-pool"
  node_class_name = "production-class"

  aws = {
    role = "KarpenterNodeRole-production"
    # Secure IMDS v2 defaults applied automatically:
    # - metadata_options.http_endpoint = "enabled"
    # - metadata_options.http_protocol_ipv6 = "disabled"
    # - metadata_options.http_put_response_hop_limit = 2
    # - metadata_options.http_tokens = "required"
  }
}

# Comprehensive AWS example with all common options
resource "devzero_node_policy" "aws_comprehensive" {
  name            = "aws-comprehensive"
  description     = "Production-ready AWS node policy with comprehensive configuration"
  node_pool_name  = "production-general-pool"
  node_class_name = "production-general-class"
  weight          = 15

  # Instance selection - Use modern, cost-effective instances
  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["c", "m", "r"] # Compute, general purpose, memory optimized
    }]
  }

  instance_generations = {
    match_expressions = [{
      key      = "instanceGenerations"
      operator = "Gt"
      values   = ["5"] # Modern instances only
    }]
  }

  # Architecture and capacity
  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["amd64"] # Can also use ["arm64"] for Graviton
    }]
  }

  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["spot", "on-demand"] # Cost optimization with fallback
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  # Node labels and taints
  labels = {
    "workload-type" = "general"
    "managed-by"    = "karpenter"
  }

  taints = [
    {
      key    = "workload-type"
      value  = "batch"
      effect = "NoSchedule"
    }
  ]

  # Disruption policy for cost optimization
  disruption = {
    consolidate_after       = "15m"
    consolidation_policy    = "WhenEmptyOrUnderutilized"
    expire_after            = "720h" # 30 days
    ttl_seconds_after_empty = 300    # 5 minutes

    budgets = [
      {
        reasons = ["Underutilized", "Empty"]
        nodes   = "10%"
      }
    ]
  }

  # Resource limits
  limits = {
    cpu    = "1000"
    memory = "1000Gi"
  }

  # AWS-specific configuration
  aws = {
    role             = "KarpenterNodeRole-production"
    instance_profile = "KarpenterNodeInstanceProfile-production"
    ami_family       = "AL2023"

    # AMI selection
    ami_selector_terms = [
      {
        alias = "al2023@latest"
      }
    ]

    # Subnet selection
    subnet_selector_terms = [
      {
        tags = {
          "karpenter.sh/discovery" = "production-cluster"
        }
      }
    ]

    # Security group selection
    security_group_selector_terms = [
      {
        tags = {
          "karpenter.sh/discovery" = "production-cluster"
        }
      }
    ]

    # Instance tags
    tags = {
      "Environment" = "production"
      "ManagedBy"   = "Karpenter"
      "Team"        = "platform"
    }

    # Block device configuration
    block_device_mappings = [
      {
        device_name = "/dev/xvda"
        ebs = {
          volume_size           = "100Gi"
          volume_type           = "gp3"
          encrypted             = true
          delete_on_termination = true
        }
      }
    ]

    # Instance store policy
    instance_store_policy = "RAID0"

    # Monitoring and networking
    detailed_monitoring         = true
    associate_public_ip_address = false

    # IMDS v2 configuration (secure defaults)
    metadata_options = {
      http_endpoint               = "enabled"
      http_protocol_ipv6          = "disabled"
      http_put_response_hop_limit = 2
      http_tokens                 = "required" # Enforces IMDSv2
    }

    # User data for custom initialization
    user_data = <<-EOT
      #!/bin/bash
      echo "Custom initialization script"
      # Add your custom setup here
    EOT
  }
}

# Azure example
resource "devzero_node_policy" "azure_example" {
  name            = "azure-production"
  description     = "Production-ready Azure node policy"
  node_pool_name  = "production-pool"
  node_class_name = "production-class"
  weight          = 10

  # Instance selection
  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["D", "E"] # General purpose, memory optimized
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
      values   = ["spot", "on-demand"]
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  # Labels
  labels = {
    "dedicated" = "karpenter"
  }

  # Disruption policy
  disruption = {
    consolidate_after    = "5m"
    consolidation_policy = "WhenEmptyOrUnderutilized"
    expire_after         = "168h" # 7 days

    budgets = [
      {
        reasons = ["Empty", "Drifted", "Underutilized"]
        nodes   = "20%"
      }
    ]
  }

  # Azure-specific configuration
  azure = {
    vnet_subnet_id  = "/subscriptions/xxx/resourceGroups/yyy/providers/Microsoft.Network/virtualNetworks/zzz/subnets/aaa"
    os_disk_size_gb = 128
    image_family    = "Ubuntu2204"
    fips_mode       = "Disabled"
    max_pods        = 110

    tags = {
      "Environment" = "production"
      "ManagedBy"   = "Karpenter"
    }
  }
}
