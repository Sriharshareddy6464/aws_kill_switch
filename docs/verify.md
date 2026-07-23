# Verification Phase Guide (`aws-kill verify`)

The `verify` command is the final audit check of the AWS Kill Switch pipeline. It ensures that the resources targeted in the plan were successfully deleted from your AWS account, alerts you if a planned destruction was missed, and lists unselected/default resources.

---

## Command Definition

```bash
go run . verify
```

---

## Execution Logic & Scenarios

The command executes differently depending on the state of the local plan and result files:

### Scenario A: Kill Action Was Missed
If a plan file (`reports/plan.json`) exists, but the destruction result file (`reports/result.json`) does not exist, the command alerts the user and prints detailed lists:

1.  **Missed Deletion Alert**:
    `Error: Generated deletion plan isn't executed. Kindly run "go run . kill"`
2.  **You have missed (Planned List)**:
    Lists all services and names that were planned for deletion but not yet executed.
3.  **You still remaining with (Unselected List)**:
    *   **Scanned & Unselected Workloads**: User-created resources found in the scan phase that were not checked/selected during the interactive TUI planning stage. If no unselected resources remain, it prints: `No active resources remain outside the current deletion plan.`
    *   **Default AWS Resources (Protected)**: Discovers and prints default VPCs, default subnets, and default security groups present in the account, clearly marking them as `Default / Protected` so that the developer is aware they are kept safe.

---

### Scenario B: Normal Verification Path (Kill Action Executed)
If `reports/result.json` exists, the command runs a post-cleanup audit check:

1.  **Audit Queries**:
    *   Loops through all resources recorded as successfully deleted in `reports/result.json`.
    *   Directly queries AWS APIs (e.g. `DescribeInstances`, `ListBuckets`) to confirm that these resource IDs are no longer present (expecting `404 Not Found` or empty results).
2.  **Generate Audit Report**:
    *   Writes a final report to `reports/verification.json`.
    *   The report contains `verified: true` if all targeted resources are successfully verified as deleted. If any resources managed to survive, they are listed under `remaining_resources` for manual inspection.
