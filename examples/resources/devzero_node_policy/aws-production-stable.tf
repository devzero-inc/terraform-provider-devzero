# Production-Stable AWS Node Policy
#
# This example demonstrates a node policy optimized for stability:
# - Conservative consolidation (2 hours)
# - On-demand instances prioritized
# - Long node expiry (30 days)
# - Small disruption budget (10%)
# - Detailed monitoring enabled

resource "devzero_node_policy" "production_stable" {
  name            = "production-stable"
  description     = "Production-grade stable node policy for critical workloads"
  node_pool_name  = "production-stable-pool"
  node_class_name = "production-stable-class"
  weight          = 20 # Higher priority

  # Instance selection - proven, stable instances
  instance_categories = {
    match_expressions = [{
      key      = "instanceCategories"
      operator = "In"
      values   = ["c", "m", "r"] # Broad selection for availability
    }]
  }

  instance_generations = {
    match_expressions = [{
      key      = "instanceGenerations"
      operator = "Gt"
      values   = ["4"] # Proven generations
    }]
  }

  # Use amd64 for maximum compatibility
  architectures = {
    match_expressions = [{
      key      = "architectures"
      operator = "In"
      values   = ["amd64"]
    }]
  }

  # Prioritize on-demand for stability
  capacity_types = {
    match_expressions = [{
      key      = "capacityTypes"
      operator = "In"
      values   = ["on-demand", "spot"] # On-demand first, spot as fallback
    }]
  }

  operating_systems = {
    match_expressions = [{
      key      = "operatingSystems"
      operator = "In"
      values   = ["linux"]
    }]
  }

  # Label nodes as production-critical
  labels = {
    "workload-type" = "production"
    "stability"     = "high"
    "sla"           = "99.9"
  }

  # Conservative disruption policy for stability
  disruption = {
    consolidate_after       = "2h" # Conservative - wait 2 hours
    consolidation_policy    = "WhenEmptyOrUnderutilized"
    expire_after            = "720h" # 30 days - long-lived nodes
    ttl_seconds_after_empty = 600    # 10 minutes - don't rush termination

    budgets = [
      {
        reasons = ["Underutilized", "Empty"]
        nodes   = "10%" # Conservative budget - less disruption
        # Optional: Schedule disruption during maintenance windows
        schedule = "0 2 * * *" # 2 AM daily
        duration = "1h"        # 1 hour window
      }
    ]
  }

  # Moderate resource limits
  limits = {
    cpu    = "1000"
    memory = "2000Gi"
  }

  # AWS configuration optimized for stability
  aws = {
    role             = "KarpenterNodeRole-production"
    instance_profile = "KarpenterNodeInstanceProfile-production"
    ami_family       = "AL2" # Amazon Linux 2 - proven and stable

    ami_selector_terms = [
      {
        # Use specific AMI version for consistency
        name = "amazon-eks-node-1.28-*"
      }
    ]

    subnet_selector_terms = [
      {
        tags = {
          "karpenter.sh/discovery" = "production-cluster"
          "tier"                   = "private"
          "availability"           = "high"
        }
      }
    ]

    security_group_selector_terms = [
      {
        tags = {
          "karpenter.sh/discovery" = "production-cluster"
          "security-level"         = "production"
        }
      }
    ]

    tags = {
      "Environment" = "production"
      "CostCenter"  = "critical-services"
      "SLA"         = "99.9"
      "Backup"      = "required"
      "Monitoring"  = "enhanced"
    }

    # Generous block device for stability
    block_device_mappings = [
      {
        device_name = "/dev/xvda"
        ebs = {
          volume_size           = "200Gi" # Larger for logs and cache
          volume_type           = "gp3"
          iops                  = 3000 # Higher IOPS for performance
          throughput            = 125  # Higher throughput
          encrypted             = true
          delete_on_termination = true
        }
      }
    ]

    # Instance store for performance
    instance_store_policy = "RAID0"

    # Enable detailed monitoring for production
    detailed_monitoring         = true
    associate_public_ip_address = false

    # Secure IMDS v2 configuration
    metadata_options = {
      http_endpoint               = "enabled"
      http_protocol_ipv6          = "disabled"
      http_put_response_hop_limit = 2
      http_tokens                 = "required"
    }

    # Custom user data for production requirements
    user_data = <<-EOT
      #!/bin/bash
      set -euo pipefail

      # Enhanced logging
      exec > >(tee -a /var/log/user-data.log)
      exec 2>&1

      # Install monitoring agents
      echo "Installing CloudWatch agent..."
      wget https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm
      rpm -U ./amazon-cloudwatch-agent.rpm

      # Configure system limits for production
      cat >> /etc/security/limits.conf <<EOF
      * soft nofile 65536
      * hard nofile 65536
      * soft nproc 65536
      * hard nproc 65536
      EOF

      # Enable kernel tuning for production
      cat >> /etc/sysctl.conf <<EOF
      net.core.somaxconn = 32768
      net.ipv4.tcp_max_syn_backlog = 8192
      net.ipv4.tcp_tw_reuse = 1
      EOF
      sysctl -p

      echo "Production node initialization complete"
    EOT
  }
}

# Target this policy to production clusters
resource "devzero_cluster" "production" {
  name = "production-us-east-1"
}

resource "devzero_cluster" "production_backup" {
  name = "production-us-west-2"
}

resource "devzero_node_policy_target" "production_stable_target" {
  name        = "production-critical-clusters"
  description = "Apply stable production policy to critical workload clusters"
  policy_id   = devzero_node_policy.production_stable.id
  enabled     = true
  cluster_ids = [
    devzero_cluster.production.id,
    devzero_cluster.production_backup.id,
  ]
}
