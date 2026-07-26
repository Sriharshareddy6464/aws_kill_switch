# Project Roadmap

AWS Kill Switch roadmap outlines past accomplishments and future milestones.

---

## Phase 1: Core CLI & Scanning (Completed)
- [x] Initial codebase design and Cobra/Viper CLI command structure.
- [x] Enforce execution sequence constraints (Scan -> List -> Plan -> Kill -> Verify -> Explain).
- [x] Integrate live AWS SDK v2 resource discovery queries for all 15 services.
- [x] Implement structured status summary reports (`reports/status.json`) and inventory mappings (`reports/inventory.json`).
- [x] Build local-only `list` command to print grouped resource counts from `status.json`.

---

## Phase 2: Core Dependency Planning Engine (Completed)
- [x] Build DAG (Directed Acyclic Graph) engine in `engine/planner.go` with cyclic graph detection and breaking.
- [x] Establish topological sorting with standard service dependency rules (e.g. Subnets require Route Tables and VPCs to be clean).
- [x] Group planned resources dynamically by VPC networks or independent global classifications.
- [x] Implement tag-based filtering (`--tag`) within scanner resource discovery.

---

## Phase 3: Interactive TUI & Deletion Engine (Completed)
- [x] Build interactive checkbox checklist using `survey` to let developers selectively plan resources.
- [x] Connect AWS service modules to SDK delete APIs.
- [x] Implement sequential dependency deletion loops with dot-loader progress bars.
- [x] Add global dry-run execution checks (`--dry-run`).
- [x] Implement polling waiters with backing retry strategies for eventual consistency.

---

## Phase 4: Verification & Troubleshooting (Completed)
- [x] Refactor `Verifier` logic to query live AWS APIs in real time, auditing cloud resource IDs directly instead of trusting local receipt logs.
- [x] Style verification terminal output with progressive fading dot loading animations and line-by-line typewriter output presentation.
- [x] Add ANSI color-coded lists separating `Successfully Deleted` from `Failed Termination` groups.
- [x] Style verification summary statuses based on final check outcomes: green success, orange blinking partial success, and bright red total failure.
- [x] Implement `explain` subcommand to diagnose verify blocker states (S3 non-empty, Security Group rules) and output exact resolution instructions.
- [x] Handle CloudFront stateful lifecycle transitions by automatically disabling distributions, waiting for propagation, and outputting custom diagnostic steps.

---

## Future Milestones
- [ ] Add support for multi-region simultaneous scanning.
- [ ] Implement resource restoration configurations (backing up config maps before deletion).
- [ ] Introduce `--retry` execution recovery options directly inside the `kill` command.
