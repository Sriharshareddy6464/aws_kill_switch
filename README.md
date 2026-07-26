# AWS Kill Switch (`aws-kill`)

A lightweight, dependency-aware command-line utility in Go designed to automate the discovery, planning, sequencing, and destruction of temporary AWS development infrastructure. 

It eliminates the tedious task of manually cleaning up interconnected cloud resources (VPCs, Subnets, EC2 Instances, ALBs, RDS databases, S3 buckets, CloudFront distributions, etc.) when sandbox environments or dev tests are no longer needed.

> [!IMPORTANT]
> **Safety and Scope Boundaries**:
> This tool is strictly a developer utility meant for **sideprojects cleanups**. To prevent accidental destruction or disruption:
> 1. It **excludes all AWS default infrastructure** (default VPCs, default subnets, default security groups) at the scan level.
> 2. It **requires explicit interactive confirmation** and grouping selection at the planning stage.
> 3. It provides a **`--dry-run` flag** to simulate all deletions safely before execution.

> [!WARNING]
> **AWS Kill Switch permanently deletes AWS infrastructure.**
> This project is intended for development and educational environments only.
> It has not been designed or tested for production workloads.
> Always review the generated execution plan before running the **Kill** phase.
> **Use at your own risk.**

---

## System Architecture

The application is structured into four decoupled layers to separate CLI interaction, state management, orchestration logic, and AWS API connectivity:

```
                  ┌────────────────────────────────────────┐
                  │          Cobra CLI Cmd Layer           │
                  │ (scan, list, plan, kill, verify, explain)
                  └───────────────────┬────────────────────┘
                                      │
                                      ▼
                  ┌────────────────────────────────────────┐
                  │      Workflow Engines (orchestrate)     │
                  │   (Scanner, Planner, Killer, Verifier) │
                  └───────────────────┬────────────────────┘
                                      │
                                      ▼
                  ┌────────────────────────────────────────┐
                  │       Unified Client Registry          │
                  │          (aws.Config Session)          │
                  └───────────────────┬────────────────────┘
                                      │
                                      ▼
                  ┌────────────────────────────────────────┐
                  │          AWS Service Modules           │
                  │      (ec2.go, rds.go, cloudfront.go...)│
                  └────────────────────────────────────────┘
```

*   **CLI Cmd Layer (`cmd/`)**: Handles arguments, flags, interactive terminal UI prompts, typewriter reports, and local state file validations.
*   **Workflow Engines (`engine/`)**: Implements topological sorting, graph cycle breaking, sequential execution loops, and active waiters.
*   **AWS Service Modules (`services/`)**: Individual service wrapper files managing SDK inputs/outputs.

---

## Workflow Diagram

To prevent state inconsistencies, `aws-kill` enforces a strict sequence of command transitions:

```
      [scan] ────> [reports/status.json] ────> [list] (offline view)
        │
        ▼
[reports/inventory.json] ────> [plan] (interactive selection & confirm)
                                  │
                                  ▼
                          [reports/plan.json] ────> [kill] ────> [reports/result.json]
                                                      │
                                                      ▼
                                                   [verify]
                                                      │
                                                      ▼
                                              [reports/verification.json]
                                                      │
                                                      ▼
                                                   [explain]
```

*   **`scan`**: Performs live API reads to discover resources and writes the local inventory status.
*   **`list`**: Offline read of the scan status summary.
*   **`plan`**: Groups resources by VPC network, shows an interactive TUI picker, and records the validated deletion sequence.
*   **`kill`**: Sequentially terminates the selected resource plan and polls active states. Handles CloudFront auto-disabling lifecycle transitions.
*   **`verify`**: Confirms that target resource IDs are no longer present in AWS.
*   **`explain`**: Translates verify failures/existing resources into developer troubleshooting fixes.

---

## Installation & Setup Guide

### 1. Prerequisites
*   **Go**: Install **Go 1.24+** on your local machine.
*   **AWS CLI**: The utility executes API requests authenticated via your local AWS CLI credentials. Install the official AWS CLI and configure access keys before proceeding:
    ```bash
    aws configure
    ```
    Enter your Access Key ID, Secret Access Key, target region, and output format.

### 2. Compile & Install
To build the application and package it as a native executable:

#### Windows Users:
Run the automated packaging installer:
```cmd
build.bat
```
This compiles the code into `aws-kill.exe` and displays instructions on how to add it to your System PATH so you can run `aws-kill` from any folder.

#### macOS / Linux Users:
```bash
go build -o aws-kill
```

---

## How To Use (Step-by-Step Walkthrough)

Follow these steps sequentially to scan, plan, destroy, verify, and troubleshoot your AWS infrastructure:

### Step 0: Configure AWS Authentication
Ensure your environment is configured to point to your target sandbox account. You can configure credentials using:
```bash
aws configure
```
To run the tool using a specific profile, use the `--profile` flag.

### Step 1: Scan Active Resources
Scan your AWS account to discover all active, non-default workloads. You can specify AWS CLI profile, region, or tags:
```bash
aws-kill scan --profile my-dev-profile --region us-east-1 --tag Project=sandbox-app
```
*Creates: `reports/inventory.json` (details of resources found) and `reports/status.json` (offline list summaries)*

### Step 2: List Discovered Resources Offline
View the scan summaries offline without query charges or active AWS connections:
```bash
aws-kill list
```

### Step 3: Create an Interactive Deletion Plan
Launch the interactive planner. It groups your infrastructure by VPC network groups and global services, presenting a checkbox selection overlay. Use the Arrow Keys to scroll, Spacebar to toggle VPC groups, and Enter to confirm:
```bash
aws-kill plan
```
*Creates: `reports/plan.json` (topologically sorted deletion sequence)*

### Step 4: Execute Cleanup (Simulated Dry-Run)
Ensure the planned sequence is correct and safe to run:
```bash
aws-kill kill --dry-run
```

### Step 5: Execute Cleanup (Live Destruction)
Run the live termination engine. Deletions are sequenced topologically and polled for eventual consistency (e.g. waiting for EC2 instances to terminate and subnets to detach):
```bash
aws-kill kill
```
*Creates: `reports/result.json`*

### Step 6: Verify Deletion Status (Live Audit)
Query live AWS APIs in real time to verify that planned resource IDs are completely removed:
```bash
aws-kill verify
```
*Plays starting dot loading animations and writes: `reports/verification.json`*

### Step 7: Troubleshoot Blockages
If the verification phase reports remaining resources (Status: `PARTIAL SUCCESS`), run `explain` to retrieve detailed diagnostics and actionable resolutions:
```bash
aws-kill explain
```
*Outputs specific instructions (like `aws s3 rb s3://<bucket> --force`) to resolve residual blocks.*
