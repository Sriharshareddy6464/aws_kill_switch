# Verification Phase Guide (`aws-kill verify`)

The `verify` command is the final audit check of the AWS Kill Switch pipeline. It ensures that the resources targeted in the plan were successfully deleted from your AWS account and generates a final verification report.

---

## Command Definition

```bash
go run . verify
```

---

## Execution Logic

1.  **Guard Check**: Verifies that `reports/result.json` exists, confirming that a `kill` run was attempted.
2.  **Audit Queries**:
    *   Loops through all resources recorded as successfully deleted in `reports/result.json`.
    *   Directly queries AWS APIs (e.g. `DescribeInstances`, `ListBuckets`) to confirm that these resource IDs are no longer present (expecting `404 Not Found` or empty results).
3.  **Generate Audit Report**:
    *   Writes a final report to `reports/verification.json`.
    *   The report contains `verified: true` if all targeted resources are successfully verified as deleted. If any resources managed to survive, they are listed under `remaining_resources` for manual inspection.
