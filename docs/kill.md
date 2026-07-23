# Destruction Phase Guide (`aws-kill kill`)

The `kill` command executes the deletion order. It sequentially calls AWS delete APIs for each resource and polls AWS to verify resource termination before proceeding.

---

## Command Definition

```bash
go run . kill [flags]
```

### Supported Flags

*   `--dry-run`: Runs the complete execution flow in simulated mode. It logs all actions and writes simulated successes to `reports/result.json` without executing any real AWS API write/delete operations. (Highly recommended for initial testing).

---

## Execution Logic

1.  **Read Plan**: Loads the targeted deletion list from `reports/plan.json`.
2.  **Sequential Destruction Dispatcher**: Loops through resources in the exact plan order and invokes the corresponding service's delete method.
3.  **Active Wait & Polling (Eventual Consistency)**:
    *   AWS deletions are asynchronous and eventually consistent. For example, detaching an Internet Gateway or terminating an EC2 instance can take time.
    *   For asynchronous resource types (EC2, EBS Volumes, NAT Gateways, ALBs, RDS), the killer calls the `aws/waiters.go` module to poll status (e.g. checking until instance state is `terminated` or the resource ID returns a `404 Not Found` error) before proceeding to the next step.
4.  **Save Outcomes**: Writes execution results to `reports/result.json` (listing successful and failed operations).

---

## Verification & Error Resilience

*   If a deletion step fails (e.g. a timeout or eventual consistency delay blocks a subnet from deleting), the killer logs the error to `stderr`, records it under `failed_resources` in `reports/result.json`, and **continues** processing the remaining independent branches.
*   The final execution summary is printed showing the counts of deleted and failed resources.
