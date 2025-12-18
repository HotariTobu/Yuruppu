# OpenTofu Language Overview

The OpenTofu language is the primary user interface for infrastructure automation. Its main purpose is declaring resources, which represent infrastructure objects, with all other features designed to enhance resource definition flexibility.

## Fundamental Syntax Elements

OpenTofu configurations use three basic building blocks:

### 1. Blocks

Containers representing configuration objects (like resources), containing a type, optional labels, and a body with arguments and nested blocks.

```hcl
resource "aws_instance" "example" {
  ami           = "ami-abc123"
  instance_type = "t2.micro"
}
```

### 2. Arguments

Name-value pairs appearing within blocks that assign values to configuration elements.

```hcl
ami           = "ami-abc123"
instance_type = "t2.micro"
```

### 3. Expressions

Values representing either literals or combinations of other values, used in arguments and other expressions.

```hcl
instance_type = var.instance_type
name          = "${var.environment}-server"
count         = length(var.availability_zones)
```

## Language Characteristics

### Declarative Approach

The language employs a declarative approach, describing an intended goal rather than the steps to reach that goal.

Resource ordering and file organization are generally insignificant—OpenTofu determines operation sequence based on implicit and explicit resource relationships.

### File Structure

Configuration files use:
- `.tf` extension for HCL syntax
- `.tofu` extension for OpenTofu-specific files
- `.tf.json` or `.tofu.json` for JSON syntax

All files in a directory are loaded together, so you can organize your configuration across multiple files as you see fit.

## Key Language Elements

### Resources

The most important element. Each resource block describes one or more infrastructure objects.

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/my-project/api:latest"
      }
    }
  }
}
```

### Data Sources

Read information from external sources or existing infrastructure.

```hcl
data "google_project" "current" {}

data "google_secret_manager_secret_version" "api_key" {
  secret = "api-key"
}
```

### Providers

Configure provider plugins that interact with cloud platforms.

```hcl
provider "google" {
  project = "my-gcp-project"
  region  = "us-central1"
}
```

### Variables

Define inputs to make configurations reusable.

```hcl
variable "environment" {
  type        = string
  description = "Deployment environment"
}
```

### Outputs

Export values for use by other configurations or display after apply.

```hcl
output "service_url" {
  value = google_cloud_run_service.api.status[0].url
}
```

### Locals

Define intermediate computed values used within a module.

```hcl
locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "OpenTofu"
  }
}
```

### Modules

Group related resources for reusability.

```hcl
module "network" {
  source = "./modules/vpc"

  project_id = var.project_id
  region     = var.region
}
```

## Meta-Arguments

Special arguments available across all resource types:

- `depends_on` - Explicit dependencies
- `count` - Create multiple instances based on a count
- `for_each` - Create instances from maps or sets
- `provider` - Select non-default provider configuration
- `lifecycle` - Customize resource lifecycle behavior

## Configuration Structure

Complete configurations combine multiple elements: provider declarations, variable definitions, resource blocks, and module references—all working together to define managed infrastructure.

A typical structure:

```
main.tf           # Main resource definitions
variables.tf      # Input variable declarations
outputs.tf        # Output value declarations
providers.tf      # Provider configurations
versions.tf       # Version constraints
terraform.tfstate # State file (generated)
```

## Comments

```hcl
# Single-line comment

/*
  Multi-line
  comment
*/
```

## Best Practices

1. Use descriptive resource names
2. Organize code into logical files
3. Use modules for reusable components
4. Define type constraints for variables
5. Document outputs and variables with descriptions
6. Use locals for computed values used multiple times
7. Keep provider configurations separate
8. Version control your configurations
