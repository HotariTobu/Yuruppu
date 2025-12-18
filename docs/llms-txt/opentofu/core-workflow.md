# OpenTofu Core Workflow

The fundamental OpenTofu workflow consists of three primary steps: Write, Plan, and Apply. This core workflow is a loop—teams and individuals repeat these steps for each infrastructure change.

## 1. Write

Author infrastructure as code in your editor of choice, typically stored in version-controlled repositories.

Developers compose resource definitions across multiple cloud providers using OpenTofu's declarative configuration language. You iteratively run plans to catch syntax errors and verify configuration progress.

**Individual practitioners** follow this workflow directly with local execution and straightforward version control commits.

**Teams** introduce additional complexity by using version control branches and incorporating code review processes during this phase.

## 2. Plan

Preview changes before applying. The `tofu plan` command displays proposed infrastructure modifications for review.

```bash
tofu plan
```

This command:
- Shows what resources will be created, updated, or destroyed
- Validates your configuration syntax
- Provides a detailed execution plan
- Does not modify actual infrastructure

For teams, this creates a checkpoint where colleagues can evaluate the proposed changes during pull request reviews before any real resources are affected.

### Saving Plans

You can save plans to files for later execution:

```bash
tofu plan -out=tfplan
```

## 3. Apply

Provision reproducible infrastructure after final confirmation. Running `tofu apply` executes the planned changes to your actual environment.

```bash
tofu apply
```

OpenTofu will:
1. Show you the execution plan one final time
2. Ask for confirmation (unless `-auto-approve` is used)
3. Execute the changes in proper dependency order
4. Update the state file to reflect the new infrastructure

### Applying Saved Plans

If you saved a plan file, apply it directly:

```bash
tofu apply tfplan
```

This executes the exact plan without showing a new preview or asking for confirmation.

## Team vs Individual Workflows

### Individual Practitioners

- Execute commands locally
- Commit changes directly to version control
- Manage state files locally or in simple backends

### Team Workflows

Teams often:
- Use version control branches for each change
- Require code review via pull requests before applying
- Migrate to Continuous Integration environments where OpenTofu operations run centrally rather than on individual machines
- Store state in remote backends (like GCS) for collaboration
- Use state locking to prevent concurrent modifications

This approach mitigates security risks associated with storing sensitive credentials locally.

## Iteration Frequency

The frequency of repeating this workflow varies by organizational needs—some teams deploy multiple times daily, while others deploy weekly or monthly.
