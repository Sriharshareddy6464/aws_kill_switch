# Listing Phase Guide (`aws-kill list`)

The `list` command provides an immediate, offline summary of active resources discovered in the latest scan. It presents a categorized summary of resource counts.

---

## Command Definition

```bash
go run . list
```

---

## Technical Design & Zero-API Execution

A key design feature of the `list` command is that **it does not make any live network calls to AWS APIs.**

1.  **Reads Status Artifacts**: It reads the `reports/status.json` file generated in the preceding `scan` phase.
2.  **No AWS Config Required**: Because it runs entirely offline, it does not require AWS CLI profiles, active internet connections, or credential validation.
3.  **Strict State Transition Guard**:
    *   If `reports/status.json` does not exist (meaning a scan was never run), the command exits immediately with error code `1` and prints:
        `Error: No scan status report found at reports/status.json. Please run 'aws-kill scan' first.`

---

## Formatted Console Output

The command reads the aggregated counts and displays them grouped under their corresponding AWS services (e.g., EC2, VPC, ALB, ECS, RDS, S3, CloudFront).

### Deduplication Logic
To prevent misleading metrics, the total summary calculation at the bottom of the output **deduplicates nested counts** and excludes derived states:
*   `Running Instances`, `Stopped Instances` (derived from total `Instances`) are not double-counted.
*   `Images` in ECR and `Running Tasks` in ECS are skipped from the raw summation of target entities to be deleted.

This ensures the user sees an accurate overview of parent resources.
