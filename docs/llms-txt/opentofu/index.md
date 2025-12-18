# OpenTofu

> OpenTofu is an open-source infrastructure-as-code tool that lets you define cloud and on-premises resources in declarative configuration files. It manages infrastructure through providers, maintains state, and enables reproducible infrastructure deployments. This documentation covers OpenTofu v1.10 for managing GCP infrastructure (Cloud Run, Artifact Registry, Secret Manager, IAM, Cloud Build).

OpenTofu is a community-driven fork that maintains compatibility with Terraform while providing open governance. It uses a declarative language to describe infrastructure, automatically determines resource dependencies, and provisions resources in parallel where possible.

## Getting Started

- [Introduction](introduction.md): What OpenTofu is, core concepts, and how it works
- [Installation](installation.md): Installation methods across platforms
- [Core Workflow](core-workflow.md): Write, Plan, and Apply workflow
- [GCS Backend](gcs-backend.md): Configure Google Cloud Storage backend for state storage

## Language Reference

- [Language Overview](language/overview.md): Configuration language fundamentals and syntax
- [Resources](language/resources.md): Declaring and managing infrastructure resources
- [Data Sources](language/data-sources.md): Reading external information and computed values
- [Providers](language/providers.md): Configuring and using providers
- [Variables](language/variables.md): Input variable declaration and usage
- [Outputs](language/outputs.md): Output value declaration and usage
- [Modules](language/modules.md): Module structure and reusability
- [State Management](language/state.md): Understanding and managing state
- [Expressions](language/expressions.md): Reference syntax and value access

## Meta-Arguments

- [count](language/count.md): Creating multiple resource instances with count
- [for_each](language/for-each.md): Creating multiple instances from maps or sets
- [lifecycle](language/lifecycle.md): Controlling resource lifecycle behavior

## CLI Commands

- [init](cli/init.md): Initialize a working directory
- [plan](cli/plan.md): Preview infrastructure changes
- [apply](cli/apply.md): Execute infrastructure changes
- [destroy](cli/destroy.md): Destroy managed infrastructure
- [import](cli/import.md): Import existing resources into state
- [state](cli/state.md): Advanced state management

## Configuration

- [Environment Variables](config/environment-variables.md): Configure OpenTofu via environment variables

## Optional

- [Module Sources](language/module-sources.md): Local paths, Git, registry, and cloud storage sources
- [Migration Guide](https://opentofu.org/docs/v1.10/intro/migration/): Migrating from Terraform
- [What's New](https://opentofu.org/docs/v1.10/intro/whats-new/): Latest features in v1.10
- [Public Registry](https://registry.opentofu.org/): Browse providers and modules
- [Full Documentation](https://opentofu.org/docs/v1.10/): Complete online documentation
