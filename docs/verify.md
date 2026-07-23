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
2.  **TUI Loading Animations**: On startup, progressive LED-wave dot loops (fading in one by one at 500ms intervals) start playing instantly to avoid a blank screen freeze:
    *   `retrieving ......`
    *   `processing ......`
    *   `organising .....`
3.  **Background AWS Caching**: While the loading screen animations are running, live AWS checks are executed concurrently in a separate thread. This hides API latency so the report appears instantly as soon as `organising` finishes.
4.  **Typewriter Report Presentation**: Once loaded, the verification dashboard is printed line-by-line with an 80ms typing delay to provide a dynamic console feedback effect.
5.  **Verified Representation Layout**: Outputs two separate lists styled with ANSI escape color codes:
    *   **Successfully Deleted** (green bold heading): Column-aligned listing of all verified deleted resources prefixed with a green tick (`✓`).
    *   **Failed Termination** (red bold heading): Listing of remaining active workloads prefixed with a red cross (`✗`) and suffixed with a red `(Still Exists)` or `(Verification Failed)` details.
6.  **Generate Audit Report**: Writes a JSON report to `reports/verification.json` listing the verification status (`DELETED`, `EXISTS`, or `FAILED`) for each resource.
7.  **Display Summary**: Prints the counts of planned, verified deleted, still existing, and error items. The `Verification Status` is styled dynamically:
    *   **ALL TERMINATION SUCCESS** (green): If all resources are verified as successfully deleted.
    *   **PARTIAL SUCCESS** (blinking orange/yellow): If some resources were deleted but others still remain active.
    *   **FAILED TERMINATION** (bold bright red): If all planned resources failed to delete.
8.  **Exit Status**: The command always returns clean exit code `0` to signal verification checks executed successfully, even if some resources still remain active.
