# Verification Phase Guide (`aws-kill verify`)

The `verify` command is the final audit check of the AWS Kill Switch pipeline. It queries live AWS APIs in real time to verify that the target resource IDs listed in `reports/plan.json` are completely removed.

---

## Command Definition

```bash
go run . verify
```

---

## Execution Logic (Architecture Design)

Unlike other steps, verification does not trust local receipt files (such as `reports/result.json`). It compares expected future state (from `plan.json`) against actual cloud reality (from live AWS APIs):

```text
plan.json (Expected State)
     │
     ▼
[verify command] ───(Live API Queries)───> AWS Infrastructure (Actual State)
     │
     ▼
reports/verification.json (Audit Report)
```

1.  **Read Plan**: Loads expected targets from `reports/plan.json`. If it doesn't exist, it prints instructions to scan/plan and exits.
2.  **TUI Loading Animations**: On startup, a 2-second delay is introduced. It then plays sequential LED-wave dot progressions:
    *   `retrieving ......` (dots fade in one by one at 500ms intervals)
    *   `processing ......` (dots fade in one by one at 500ms intervals)
    *   `organising .....` (dots fade in one by one at 500ms intervals)
3.  **Background AWS Caching**: While the loading screen animations are running, live AWS checks are executed concurrently in a separate thread. This hides API latency so the report appears instantly as soon as `organising` finishes.
4.  **Verified Representation Layout**: Outputs two separate lists:
    *   **successfully deleted**: Grouped and column-aligned listing of all verified deleted resources.
    *   **failed deletion**: Listing of remaining active workloads with `(Still Exists)` or `(Verification Failed)` details.
5.  **Generate Audit Report**: Writes a JSON report to `reports/verification.json` listing the verification status (`DELETED`, `EXISTS`, or `FAILED`) for each resource.
6.  **Display Summary**: Prints the counts of planned, verified deleted, still existing, and error items.
7.  **Exit Status**: The command always returns clean exit code `0` to signal verification checks executed successfully, even if some resources still remain active.
