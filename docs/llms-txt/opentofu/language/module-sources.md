# Module Sources

OpenTofu supports multiple methods for specifying module sources in the `source` argument of a module block. This allows you to load modules from various locations.

## Local Paths

Local references use relative paths beginning with `./` or `../`. Local paths are not 'installed' in the same sense that other sources are since files already exist on disk.

```hcl
module "cloud_run" {
  source = "./modules/cloud-run"

  project_id = var.project_id
  region     = var.region
}
```

```hcl
module "shared_network" {
  source = "../shared-modules/network"

  project_id = var.project_id
}
```

**Use for:**
- Project-specific modules
- Shared modules within an organization
- Development and testing

## Module Registry

The native distribution method uses the format `<NAMESPACE>/<NAME>/<PROVIDER>`. For private registries, add `<HOSTNAME>/` as a prefix.

### Public Registry

```hcl
module "gke" {
  source  = "terraform-google-modules/kubernetes-engine/google"
  version = "~> 30.0"

  project_id = var.project_id
  name       = "my-cluster"
  region     = var.region
}
```

### Private Registry

```hcl
module "custom" {
  source  = "registry.company.com/namespace/module/google"
  version = "1.0.0"

  project_id = var.project_id
}
```

**Version constraints:**

```hcl
version = "1.0.0"      # Exact version
version = "~> 1.0"     # >= 1.0.0, < 2.0.0
version = ">= 1.0.0"   # At least 1.0.0
version = "~> 1.2.0"   # >= 1.2.0, < 1.3.0
```

## Version Control Systems

### GitHub

Recognized automatically without prefixes:

```hcl
module "cloud_run" {
  source = "github.com/myorg/terraform-modules//cloud-run"
}
```

**With specific ref:**

```hcl
module "cloud_run" {
  source = "github.com/myorg/terraform-modules//cloud-run?ref=v1.0.0"
}
```

**ref can be:**
- Tag: `?ref=v1.0.0`
- Branch: `?ref=main`
- Commit: `?ref=abc123def`

**Private repositories:**

```bash
# Use SSH
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

```hcl
module "private" {
  source = "github.com/myorg/private-modules//network"
}
```

### Generic Git

Use `git::` prefix with any valid Git URL:

```hcl
module "network" {
  source = "git::https://example.com/terraform-modules.git//network"
}
```

**With SSH:**

```hcl
module "network" {
  source = "git::ssh://git@gitlab.com/myorg/modules.git//network"
}
```

**With ref:**

```hcl
module "network" {
  source = "git::https://example.com/modules.git//network?ref=v2.0.0"
}
```

**Shallow clone (faster):**

```hcl
module "network" {
  source = "git::https://example.com/modules.git//network?ref=v2.0.0&depth=1"
}
```

### Bitbucket

Automatically interpreted for public repositories:

```hcl
module "cloud_run" {
  source = "bitbucket.org/myorg/terraform-modules//cloud-run"
}
```

### Mercurial

Use `hg::` prefix:

```hcl
module "network" {
  source = "hg::https://example.com/modules//network"
}
```

**With ref:**

```hcl
module "network" {
  source = "hg::https://example.com/modules//network?ref=v1.0"
}
```

## Cloud Storage

### S3 Bucket

Use `s3::` prefix with S3 object URLs:

```hcl
module "cloud_run" {
  source = "s3::https://s3.amazonaws.com/my-modules/cloud-run.zip"
}
```

**With specific region:**

```hcl
module "cloud_run" {
  source = "s3::https://s3-eu-west-1.amazonaws.com/my-modules/cloud-run.zip"
}
```

### GCS Bucket

Use `gcs::` prefix with Google Cloud Storage URLs:

```hcl
module "cloud_run" {
  source = "gcs::https://www.googleapis.com/storage/v1/my-bucket/modules/cloud-run.zip"
}
```

**Authentication:**
- Uses GOOGLE_APPLICATION_CREDENTIALS
- Or Application Default Credentials

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
tofu init
```

## OCI Distribution

Use `oci://` protocol for container registries:

```hcl
module "cloud_run" {
  source = "oci://registry.example.com/modules/cloud-run:1.0.0"
}
```

**Artifact Registry:**

```hcl
module "cloud_run" {
  source = "oci://us-central1-docker.pkg.dev/my-project/modules/cloud-run:1.0.0"
}
```

## HTTP/HTTPS URLs

URLs can redirect to other source types. Archives are auto-detected by extension:

```hcl
module "cloud_run" {
  source = "https://example.com/modules/cloud-run.zip"
}
```

**Supported archive formats:**
- `.zip`
- `.tar.gz`
- `.tar.bz2`
- `.tar.xz`

**With checksum verification:**

```hcl
module "cloud_run" {
  source = "https://example.com/modules/cloud-run.zip?checksum=sha256:abc123..."
}
```

## Subdirectories

Use double-slash syntax (`//`) to reference modules within packages:

```hcl
# Git repository with multiple modules
module "vpc" {
  source = "git::https://example.com/network.git//modules/vpc"
}

module "subnet" {
  source = "git::https://example.com/network.git//modules/subnet"
}
```

```hcl
# GCS bucket with multiple modules
module "api" {
  source = "gcs::https://www.googleapis.com/storage/v1/my-bucket/terraform-modules.zip//cloud-run-api"
}
```

## Examples by Use Case

### Monorepo Structure

```
terraform-modules/
├── modules/
│   ├── cloud-run/
│   │   ├── main.tf
│   │   └── variables.tf
│   ├── network/
│   └── iam/
└── environments/
    ├── dev/
    └── prod/
```

```hcl
# From environments/dev/main.tf
module "cloud_run" {
  source = "../../modules/cloud-run"

  project_id = var.project_id
}
```

### Public Registry Modules

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

module "vpc" {
  source  = "terraform-google-modules/network/google"
  version = "~> 9.0"

  project_id   = var.project_id
  network_name = "my-vpc"

  subnets = [
    {
      subnet_name   = "subnet-01"
      subnet_ip     = "10.10.10.0/24"
      subnet_region = "us-central1"
    }
  ]
}
```

### Private GitHub Repository

```hcl
# Configure SSH access first
# ssh-keyscan github.com >> ~/.ssh/known_hosts

module "custom_network" {
  source = "github.com/myorg/terraform-modules//network?ref=v2.1.0"

  project_id = var.project_id
  region     = var.region
}
```

### GCS Bucket Modules

```hcl
module "cloud_run" {
  source = "gcs::https://www.googleapis.com/storage/v1/company-terraform-modules/modules/cloud-run-v1.0.0.zip"

  project_id   = var.project_id
  service_name = "api"
}
```

### Multi-Environment with Versions

```hcl
# development environment
module "cloud_run" {
  source = "git::https://github.com/myorg/modules.git//cloud-run?ref=main"

  environment = "dev"
}

# production environment
module "cloud_run" {
  source = "git::https://github.com/myorg/modules.git//cloud-run?ref=v1.2.0"

  environment = "prod"
}
```

## Best Practices

1. **Use versions** - Always specify versions for registry and Git modules
2. **Pin production** - Use exact versions (tags) in production
3. **Test with branches** - Use branch refs for development
4. **Local for development** - Use local paths during module development
5. **Registry for distribution** - Publish stable modules to registries
6. **Document sources** - Comment why specific sources are used
7. **Use subdirectories** - Organize related modules in single repositories
8. **Verify checksums** - Use checksums for HTTP sources
9. **Authenticate securely** - Use SSH keys or workload identity, not passwords
10. **Cache modules** - OpenTofu caches downloaded modules in `.terraform/modules`

## Module Installation

Modules are downloaded during initialization:

```bash
tofu init
```

Modules are cached in `.terraform/modules/`:

```
.terraform/
└── modules/
    ├── modules.json
    ├── cloud_run/
    └── network/
```

**Update modules:**

```bash
tofu init -upgrade
```

**Force reinstall:**

```bash
rm -rf .terraform/modules
tofu init
```

## Troubleshooting

### Module Not Found

```
Error: Module not found

Could not download module "cloud_run" (main.tf:5)
```

**Solutions:**
- Verify source URL is correct
- Check network connectivity
- Verify authentication (for private repos)
- Ensure subdirectory path exists (after `//`)

### Authentication Errors

**GitHub:**
```bash
# Use SSH
git config --global url."git@github.com:".insteadOf "https://github.com/"

# Or use personal access token
git config --global credential.helper store
```

**GCS:**
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
gcloud auth application-default login
```

### Version Conflicts

```
Error: Incompatible module version
```

**Solution:**
- Check version constraints
- Update version in module block
- Use `tofu init -upgrade`

### Slow Downloads

```bash
# Use shallow clone for Git
source = "git::https://example.com/modules.git//network?depth=1"

# Use closer mirror
# Use CDN for HTTP sources
# Enable HTTP caching
```

## Related Documentation

- [Modules Overview](modules.md)
- [Module Development](https://opentofu.org/docs/v1.10/language/modules/develop/)
- [Public Registry](https://registry.opentofu.org/)
