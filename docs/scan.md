# Scanning Phase Guide (`aws-kill scan`)

The `scan` command is the single entry point for resource discovery in the AWS Kill Switch pipeline. It queries live AWS APIs to discover resources and compiles a local snapshot of active infrastructure.

---

## Command Definition

```bash
go run . scan [flags]
```

### Supported Flags

*   `--profile`: Specifies the AWS CLI credential profile to use (default: searches environment/default profile).
*   `--region`: Targets a specific AWS region (default: searches config/default region).
*   `--tag`: Filters resources to target specific projects (e.g., `--tag Environment=dev` or `--tag Project=docco`).

---

## Scan Lifecycle & Scoping Rules

To prevent breaking default account environments or generating safety risks, the Scan phase applies strict **scoping filters**:

1.  **Excludes AWS Default Infrastructure**:
    *   **Default VPCs** (and attached default Internet Gateways) are explicitly skipped.
    *   **Default Subnets** in any Availability Zone are skipped.
    *   **Default Security Groups** (named `default`) are skipped.
    *   **Main Route Tables** are skipped.
2.  **Preserves User Workloads in Default Networks**:
    *   If a developer runs an EC2 instance, custom database, or ALB inside the default VPC or default subnets, the scanner **will discover and record them**. They will be scheduled for cleanup, while leaving the default subnet and VPC shell intact.
3.  **Generates Dual Output Files**:
    *   `reports/inventory.json`: A full list of resources including ID, Name, Type, State, and Tags. Used by the planning engine.
    *   `reports/status.json`: A human-readable aggregated summary of counts grouped by AWS service. Used by the `list` command.

---

## Technical Invocation flow

Under the hood, the execution of the `scan` command behaves as follows:

```
[CLI Command] cmd/scan.go
      │
      ▼
[Orchestrator] engine/scanner.go
      │
      ├── (Instantiates AWS Client Session config)
      ▼
[Service Scanners] services/*.go (Concurrent API Calls)
      │
      ├── DescribeInstances, DescribeVolumes, DescribeSubnets, etc.
      ▼
[Outputs] reports/inventory.json & reports/status.json
```

If any service scanner encounters a permission or authorization error (e.g., if the user does not use RDS in their account, leading to `AccessDenied`), the orchestrator catches the warning, logs it to `stderr`, and **proceeds safely** with the remaining services.
