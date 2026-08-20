# Epic: Pulumi DevZero Provider

## Epic Summary
Build a native Pulumi provider for the DevZero platform to enable Infrastructure as Code management of DevZero resources using Pulumi's modern programming model. This provider will offer type-safe SDKs in multiple languages (TypeScript, Python, Go, C#) and integrate seamlessly with existing Pulumi stacks.

## Business Value
- Enable DevZero customers to manage their infrastructure using Pulumi's modern IaC approach
- Provide type-safe, intellisense-enabled SDKs for better developer experience
- Support multi-language ecosystems beyond HCL
- Enable integration with existing Pulumi component libraries and patterns
- Facilitate GitOps workflows for DevZero resource management

## Technical Overview
The provider will implement 5 resources matching the Terraform provider:
- Clusters
- Workload Policies
- Workload Policy Targets
- Node Policies
- Node Policy Targets

## Dependencies
- DevZero API (Connect/gRPC protocol)
- Pulumi Provider Development Framework
- Generated API clients from protobuf definitions

---

## User Stories

### Task 1: Project Setup and Boilerplate
**Description:** Initialize the Pulumi provider project with proper structure and tooling

**Acceptance Criteria:**
- [ ] Provider repository created with Pulumi provider boilerplate
- [ ] Multi-language SDK generation configured (TypeScript, Python, Go, C#)
- [ ] CI/CD pipeline setup for automated builds and releases
- [ ] Documentation structure initialized
- [ ] Development environment setup guide created
- [ ] License and contribution guidelines established

**Technical Details:**
- Use Pulumi Provider Boilerplate template
- Configure GitHub Actions for multi-platform builds
- Setup semantic versioning and release automation

---

### Task 2: Provider Configuration and Authentication
**Description:** Implement the base provider with authentication and configuration options

**Acceptance Criteria:**
- [ ] Provider accepts configuration for `token`, `teamId`, and `url`
- [ ] Environment variable support for `DEVZERO_TOKEN`, `DEVZERO_TEAM_ID`, `DEVZERO_URL`
- [ ] Proper error handling for missing or invalid credentials
- [ ] Secure token handling (marked as secret in all SDKs)
- [ ] Default URL fallback to `https://dakr.devzero.io`
- [ ] Provider configuration validation implemented

**Technical Details:**
```typescript
interface ProviderConfig {
  token: pulumi.Input<string>; // secret
  teamId: pulumi.Input<string>;
  url?: pulumi.Input<string>; // defaults to https://dakr.devzero.io
}
```

---

### Task 3: API Client Integration
**Description:** Integrate with DevZero's Connect/gRPC API

**Acceptance Criteria:**
- [ ] Connect/gRPC clients generated from protobuf definitions
- [ ] Three service clients implemented: ClusterMutation, K8S, K8SRecommendation
- [ ] Bearer token authentication interceptor implemented
- [ ] Proper error handling and retries configured
- [ ] Client connection pooling and lifecycle management
- [ ] TypeScript types generated from protobuf definitions

**Technical Details:**
- Port from `internal/gen/api/v1` Go implementation
- Use Connect RPC TypeScript client
- Implement proper timeout and retry logic

---

### Task 4: Cluster Resource Implementation
**Description:** Implement the DevZero Cluster resource

**Acceptance Criteria:**
- [ ] Create cluster with name
- [ ] Read cluster by ID
- [ ] Update cluster name
- [ ] Delete cluster
- [ ] Token rotation on update (when token is empty/unknown)
- [ ] Computed fields: `id`, `token` (marked as secret)
- [ ] Import support by cluster ID
- [ ] Proper state management and drift detection

**API Operations:**
- Create: `ClusterMutationClient.CreateCluster`
- Read: `K8SServiceClient.GetCluster`
- Update: `ClusterMutationClient.UpdateCluster` (+ `ResetClusterToken` if needed)
- Delete: `ClusterMutationClient.DeleteCluster`

**Example Usage:**
```typescript
const cluster = new devzero.Cluster("my-cluster", {
  name: "production-cluster"
});
```

---

### Task 5: Workload Policy Resource Implementation
**Description:** Implement the DevZero Workload Policy resource for workload optimization

**Acceptance Criteria:**
- [ ] Create policy with all configuration options
- [ ] Support for action triggers: `on_detection`, `on_schedule`
- [ ] Support for detection triggers: `pod_creation`, `pod_update`, `pod_reschedule`
- [ ] Cron schedule validation and support
- [ ] Vertical scaling configurations (CPU, Memory, GPU, GPU VRAM)
- [ ] Horizontal scaling configuration
- [ ] Live migration enable/disable
- [ ] All tuning parameters with proper defaults
- [ ] Import support by policy ID

**Complex Fields:**
- Vertical scaling: min/max values, scale up/down percentages
- Horizontal scaling: replica ranges, utilization targets
- Multiple boolean flags for behavior control

**Example Usage:**
```typescript
const policy = new devzero.WorkloadPolicy("optimization-policy", {
  name: "gpu-optimization",
  description: "Optimize GPU workloads",
  actionTriggers: ["on_detection"],
  detectionTriggers: ["pod_creation"],
  gpuVerticalScaling: {
    minGpu: 1,
    maxGpu: 8,
    scaleUpPercent: 50,
    scaleDownPercent: 80
  }
});
```

---

### Task 6: Workload Policy Target Resource Implementation
**Description:** Implement targeting mechanism for workload policies

**Acceptance Criteria:**
- [ ] Create target with policy reference
- [ ] Label selector support (matchLabels, matchExpressions)
- [ ] Namespace and workload selectors
- [ ] Regex pattern matching for names
- [ ] Kind filter support
- [ ] Direct targeting: workload names, node groups, cluster IDs
- [ ] Priority and enabled/disabled support
- [ ] Import support by target ID

**Complex Selectors:**
```typescript
interface LabelSelector {
  matchLabels?: { [key: string]: string };
  matchExpressions?: Array<{
    key: string;
    operator: "In" | "NotIn" | "Exists" | "DoesNotExist";
    values?: string[];
  }>;
}
```

---

### Task 7: Node Policy Resource Implementation
**Description:** Implement Karpenter-integrated node provisioning policies

**Acceptance Criteria:**
- [ ] Create policy with node pool and class names
- [ ] Instance selection criteria (categories, families, sizes, CPUs, GPUs)
- [ ] Architecture and capacity type selection
- [ ] Labels and taints configuration
- [ ] Disruption policy configuration
- [ ] Resource limits configuration
- [ ] AWS-specific configurations (AMI family, instance metadata, etc.)
- [ ] Azure-specific configurations (image family, SKU)
- [ ] Raw Karpenter YAML support
- [ ] Handle missing DELETE API (remove from state only)

**Special Considerations:**
- No delete API available - implement state-only deletion
- Support both structured config and raw YAML

---

### Task 8: Node Policy Target Resource Implementation
**Description:** Implement targeting mechanism for node policies to clusters

**Acceptance Criteria:**
- [ ] Create target with policy reference
- [ ] Multiple cluster ID support
- [ ] Enable/disable functionality
- [ ] Basic metadata (name, description)
- [ ] Handle missing DELETE API (remove from state only)
- [ ] Import support by target ID

**API Considerations:**
- Create/Update use array APIs but single resource
- Read requires list + filter pattern

---

### Task 9: TypeScript SDK Polish and Examples
**Description:** Enhance TypeScript SDK with proper types, IntelliSense, and examples

**Acceptance Criteria:**
- [ ] Comprehensive TypeScript interfaces for all resources
- [ ] Enum types for fixed values (triggers, operators, etc.)
- [ ] JSDoc documentation for all properties
- [ ] Type guards and validation helpers
- [ ] Example project with common patterns
- [ ] Integration with existing Pulumi components
- [ ] Proper handling of optional vs required fields

---

### Task 10: Python SDK Generation and Examples
**Description:** Generate and polish Python SDK with proper typing

**Acceptance Criteria:**
- [ ] Python SDK generated with type hints
- [ ] Proper handling of snake_case conventions
- [ ] Enum classes for fixed values
- [ ] Docstrings for all classes and methods
- [ ] Example Python project
- [ ] Integration with Python type checkers (mypy)
- [ ] Proper handling of optional parameters

---

### Task 11: Go SDK Generation and Examples
**Description:** Generate and polish Go SDK

**Acceptance Criteria:**
- [ ] Go SDK generated with proper struct tags
- [ ] Idiomatic Go patterns implemented
- [ ] Godoc documentation
- [ ] Example Go project
- [ ] Proper nil handling for optional fields
- [ ] Integration with Go modules

---

### Task 12: C# SDK Generation and Examples
**Description:** Generate and polish C# SDK

**Acceptance Criteria:**
- [ ] C# SDK generated with proper nullable reference types
- [ ] XML documentation comments
- [ ] Proper async/await patterns
- [ ] Example C# project
- [ ] NuGet package configuration
- [ ] Integration with .NET conventions

---

### Task 13: Integration Testing Suite
**Description:** Build comprehensive integration tests

**Acceptance Criteria:**
- [ ] Test harness for all CRUD operations
- [ ] Mock API server for unit tests
- [ ] Real API integration tests (with test account)
- [ ] Cross-resource dependency tests
- [ ] Import functionality tests
- [ ] Error scenario coverage
- [ ] Performance benchmarks
- [ ] Multi-language test coverage

---

### Task 14: Documentation and Registry Publishing
**Description:** Complete documentation and publish to Pulumi Registry

**Acceptance Criteria:**
- [ ] README with quick start guide
- [ ] Full API documentation
- [ ] Migration guide from Terraform provider
- [ ] Best practices and patterns guide
- [ ] Troubleshooting guide
- [ ] Published to Pulumi Registry
- [ ] Version compatibility matrix
- [ ] Example templates in Pulumi Examples repo

---

### Task 15: CI/CD and Release Automation
**Description:** Setup automated build, test, and release pipeline

**Acceptance Criteria:**
- [ ] Automated SDK generation for all languages
- [ ] Unit and integration test execution
- [ ] Multi-platform binary builds
- [ ] Automated version bumping
- [ ] GitHub release creation
- [ ] Package publishing (npm, PyPI, Maven, NuGet)
- [ ] Pulumi Registry update automation
- [ ] Security scanning integration

---

## Definition of Done
- [ ] All resources implemented with full CRUD operations
- [ ] Multi-language SDKs generated and tested
- [ ] Comprehensive test coverage (>80%)
- [ ] Documentation complete and published
- [ ] Provider available in Pulumi Registry
- [ ] Example projects for each language
- [ ] Performance benchmarks documented
- [ ] Security review completed
- [ ] Customer beta testing completed
- [ ] GA release announced

## Technical Risks
1. **Missing DELETE APIs**: Some resources don't have delete endpoints, requiring special handling
2. **Complex Type Conversions**: Nested structures require careful type mapping
3. **API Stability**: Ensure backward compatibility as APIs evolve
4. **Multi-language Support**: Maintaining consistency across 4 language SDKs

## Success Metrics
- Provider adoption rate among DevZero customers
- Reduction in configuration errors vs manual API usage
- Time to provision resources vs UI/CLI
- Customer satisfaction scores
- Community contributions and engagement