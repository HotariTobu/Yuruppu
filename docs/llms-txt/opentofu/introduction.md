# Introduction to OpenTofu

## What is OpenTofu?

OpenTofu is an open-source infrastructure-as-code platform that lets you define both cloud and on-premises resources in human-readable configuration files that you can version, reuse, and share.

OpenTofu manages resources on cloud platforms and other services through their application programming interfaces (APIs), enabling broad platform compatibility.

## Core Workflow

OpenTofu operates through three fundamental stages:

### 1. Write

Developers compose resource definitions across multiple cloud providers. You author infrastructure as code in your editor of choice, typically stored in version-controlled repositories.

### 2. Plan

OpenTofu generates an execution roadmap showing what infrastructure changes will occur. The `tofu plan` command displays proposed infrastructure modifications for review.

For teams, this creates a checkpoint where colleagues can evaluate the proposed changes during pull request reviews before any real resources are affected.

### 3. Apply

Upon user approval, OpenTofu executes the planned modifications in proper sequence. Running `tofu apply` executes the changes to your actual environment after one last verification step.

## Key Capabilities

### Provider Ecosystem

The community has developed thousands of providers supporting platforms like AWS, Azure, GCP, Kubernetes, and GitHub, accessible through the Public OpenTofu Registry.

Providers are plugins that enable OpenTofu to interact with cloud platforms, SaaS services, and APIs. Each provider adds resource types and data sources that OpenTofu manages.

### State Tracking

OpenTofu keeps track of your real infrastructure in a state file, which acts as a source of truth for your environment. The state:

- Maps real world resources to your configuration
- Tracks metadata
- Improves performance for large infrastructures

### Parallel Resource Provisioning

By constructing a dependency graph, OpenTofu efficiently creates non-dependent resources concurrently.

### Declarative Language

The OpenTofu language is declarative, describing an intended goal rather than the steps to reach that goal. Resource ordering and file organization are generally insignificant—OpenTofu determines operation sequence based on implicit and explicit resource relationships.

## Community Support

The project is open-source and community-driven, inviting collaboration through GitHub Discussions, issue tracking, and detailed contributing guides.
