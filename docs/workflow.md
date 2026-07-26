# AWS Kill Switch Workflow & Order Enforcement

The application enforces a strict, state-machine driven execution flow. The commands must be run in the following sequence:

```
      [Scan] ──> [reports/status.json] ──> [List]
        │
        ▼
[reports/inventory.json] ──> [Plan] ──> [Kill] ──> [Verify]
```

Unless the preceding step has successfully completed and generated the required state artifact, subsequent commands will refuse to execute.

---

## 1. Flow Stages

| Stage | Command | Required Input File | Produced Output File | Description |
| :--- | :--- | :--- | :--- | :--- |
| **1. Scan** | `aws-kill scan` | *None* | `reports/inventory.json`<br>`reports/status.json` | Scans target AWS account for resources, writes raw metadata and aggregated summary status. |
| **2. List** | `aws-kill list` | `reports/status.json` | *None* | Reads the summary status and prints active AWS resource groups and counts. |
| **3. Plan** | `aws-kill plan` | `reports/inventory.json` | `reports/plan.json` | Analyzes dependencies and maps out the optimal deletion sequence. |
| **4. Kill** | `aws-kill kill` | `reports/plan.json` | `reports/result.json` | Destroys resources in order, wait, retry, and tracks execution state. |
| **5. Verify** | `aws-kill verify` | `reports/plan.json` | `reports/verification.json` | Post-deletion check confirming no planned resources remain in AWS. |

---

## 2. Enforcement Logic & Command Guards

Each command contains validation guards to prevent executing tasks out of order:

### `aws-kill scan`
*   **Action**: Connects to live AWS APIs and queries all 15 supported service endpoints.
*   **State Impact**: Cleans up any existing `reports/status.json`, `reports/plan.json`, `reports/result.json`, and `reports/verification.json` to prevent state mismatch or stale runs.
*   **Outputs**: Writes full metadata array to `reports/inventory.json` and aggregated counts to `reports/status.json`.

### `aws-kill list`
*   **Action**: Reads `reports/status.json` and formats the display output.
*   **Guard Check**: Verifies that `reports/status.json` exists in the local directory.
*   **Failure Behavior**: Aborts with code `1` and outputs:
    `Error: No scan status report found at reports/status.json. Please run 'aws-kill scan' first.`

### `aws-kill plan`
*   **Action**: Constructs the dependency DAG and does a topological sort.
*   **Guard Check**: Verifies that `reports/inventory.json` exists in the local directory and is not empty.
*   **Failure Behavior**: Aborts with code `1` and outputs:
    `Error: No scan inventory found at reports/inventory.json. Please run 'aws-kill scan' first.`
*   **Outputs**: Saves deletion sequence order list to `reports/plan.json`.

### `aws-kill kill`
*   **Action**: Iterates over the planned resources and terminates them.
*   **Guard Check**: Verifies that `reports/plan.json` exists.
*   **Failure Behavior**: Aborts with code `1` and outputs:
    `Error: No execution plan found at reports/plan.json. Please run 'aws-kill plan' first.`
*   **Outputs**: Saves termination execution status list to `reports/result.json`.

### `aws-kill verify`
*   **Action**: Queries AWS in real time for all resources listed in the planned target list to confirm deletion status.
*   **Guard Check**: Verifies that `reports/plan.json` exists.
*   **Failure Behavior**: Aborts with code `1` and outputs:
    `Error: No execution plan found at reports/plan.json. Please run 'aws-kill plan' first.`
*   **Outputs**: Writes final status confirmation list to `reports/verification.json`.

### `aws-kill explain`
*   **Action**: Parses verification results to diagnose residual active resources and generate troubleshooting steps.
*   **Guard Check**: Verifies that `reports/verification.json` exists.
*   **Failure Behavior**: Aborts with code `1` and outputs:
    `Error: No verification report found. Please run 'aws-kill verify' first.`

