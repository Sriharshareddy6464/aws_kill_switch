# AWS Kill Switch (`aws-kill`)

A lightweight, developer-focused command-line utility designed to automate the discovery, planning, sequencing, and destruction of temporary AWS development infrastructure. 

It eliminates the tedious task of manually cleaning up interconnected cloud resources (VPCs, Subnets, EC2 Instances, ALBs, RDS databases, etc.) when sandbox environments or free tiers are no longer needed.

> [!IMPORTANT]
> **Intent and Safety Scoping**:
> This tool is strictly a developer utility meant for **sandbox account cleanups**. To prevent accidental destruction or disruption:
> 1. It **excludes all AWS default infrastructure** (default VPCs, default subnets, default security groups) at the scan level.
> 2. It **requires explicit interactive confirmation** and grouping selection at the planning stage.
> 3. It provides a **`--dry-run` flag** to simulate all deletions safely before execution.

---

## System Architecture

The application is structured into four decoupled layers to separate CLI interaction, state management, orchestration logic, and AWS API connectivity:

```
                  ┌────────────────────────────────────────┐
                  │          Cobra CLI Cmd Layer           │
                  │   (scan, list, plan, kill, verify)     │
                  └───────────────────┬────────────────────┘
                                      │
                                      ▼
                  ┌────────────────────────────────────────┐
                  │      Workflow Engines (orchestrate)     │
                  │     (Scanner, Planner, Killer, etc.)   │
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
                  │        (ec2.go, rds.go, s3.go...)      │
                  └────────────────────────────────────────┘
```

*   **CLI Cmd Layer (`cmd/`)**: Handles arguments, flags, interactive terminal UI prompts, and local state file validations.
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
                          [reports/plan.json] ────> [kill] ────> [reports/result.json] ────> [verify]
```

*   **`scan`**: Performs live API reads to discover resources and writes the local inventory status.
*   **`list`**: Offline read of the scan status summary.
*   **`plan`**: Groups resources by VPC network, shows an interactive TUI picker, and records the validated deletion sequence.
*   **`kill`**: Sequentially terminates the selected resource plan.
*   **`verify`**: Confirms that target resource IDs are no longer present in AWS.

---

## Installation & Setup Guide

### 1. Prerequisites
*   **Go**: Install **Go 1.24+** on your local machine.
*   **AWS CLI**: Ensure your local environment is authenticated to your target AWS sandbox account (e.g. via `aws configure` or `~/.aws/credentials`).

### 2. Install Dependencies
Clone the repository, enter the directory, and download required packages:
```bash
go mod tidy
```

### 3. Build & Install
Compile the codebase to a native executable:
```bash
go build -o aws-kill.exe
```

---

## Command Walkthrough Guide

Follow these steps sequentially to scan and clean up your sandbox environment:

### Step 1: Scan Resources
Discover all active resources. You can limit the scope by profile, region, or tags:
```bash
./aws-kill.exe scan --profile my-dev-profile --region ap-northeast-1 --tag Project=sandbox-app
```
*Creates: `reports/inventory.json` and `reports/status.json`*

### Step 2: List Discovered Infrastructure
View a summary of active resources offline without calling AWS:
```bash
./aws-kill.exe list
```

### Step 3: Plan Deletion Order (Interactive)
Run the planner. It will group resources into network silos (VPCs) and show an interactive checkbox menu. Use the Arrow Keys to navigate, Spacebar to toggle, and Enter to select:
```bash
./aws-kill.exe plan
```
*Creates: `reports/plan.json` (only for selected resource groups)*

### Step 4: Execute Cleanup (Simulated Dry-Run)
Test the execution sequence safely without deleting anything:
```bash
./aws-kill.exe kill --dry-run
```

### Step 5: Execute Cleanup (Live Destruction)
Permanently destroy the targeted infrastructure. Since confirmation was obtained in Step 3, this runs directly:
```bash
./aws-kill.exe kill
```
*Creates: `reports/result.json`*

### Step 6: Verify Deletions
Run final check to confirm no remaining targeted resource IDs exist in AWS:
```bash
./aws-kill.exe verify
```
*Creates: `reports/verification.json`*
