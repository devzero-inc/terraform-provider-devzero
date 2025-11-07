# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The document generation tool looks for files in the following locations by default. All other *.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or are testable even if some parts are not relevant for the documentation.

* **provider/provider.tf** example file for the provider index page
* **data-sources/`full data source name`/data-source.tf** example file for the named data source page
* **resources/`full resource name`/resource.tf** example file for the named data source page

## Available Resources

### Clusters
* `devzero_cluster` - Kubernetes cluster management

### Workload Policies
* `devzero_workload_policy` - Policies for workload optimization (vertical/horizontal scaling)
* `devzero_workload_policy_target` - Attach workload policies to specific clusters

### Node Policies
* `devzero_node_policy` - Karpenter-based node provisioning policies
* `devzero_node_policy_target` - Attach node policies to specific clusters

## Node Policy Examples

The `devzero_node_policy` resource includes several comprehensive examples:

* **resource.tf** - Basic and comprehensive examples showing minimal and full configurations
* **aws.tf** - Cost-optimized configuration for batch workloads (spot instances, aggressive consolidation)
* **azure.tf** - Complete Azure examples including GPU workloads
* **complete-multi-policy.tf** - Real-world setup with multiple policies for different workload types

### Key Features Demonstrated

**Cost Optimization:**
- Spot instance usage
- Aggressive consolidation (5m)
- ARM/Graviton architecture
- Minimal block device sizes

**Production Stability:**
- On-demand instances
- Conservative consolidation (2h)
- Enhanced monitoring
- Larger disruption budgets

**Security Best Practices:**
- IMDSv2 enforcement (required)
- Encrypted EBS volumes
- Private subnets
- Security group configuration

**Multi-Cloud:**
- AWS with EC2 node classes
- Azure with AKS node classes
- Provider-specific configurations
