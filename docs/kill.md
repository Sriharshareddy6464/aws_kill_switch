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
4.  **CloudFront Stateful Deletion Workflow**:
    *   **Direct Deletion**: Attempts deletion using current ETag version.
    *   **Automatic Disable**: If deletion fails because the distribution is enabled (`DistributionNotDisabled`), the engine automatically disables it and marks the resource as **Pending** propagation.
    *   **Blockage Explanations**: If already disabled but blocked (e.g. global deployment propagation is in progress), it prints a formatted troubleshoot block detailing AWS error code, friendly reason, and next actions.
5.  **Save Outcomes**: Writes execution results to `reports/result.json` (listing successful, failed, and pending propagation operations).

## Interactive Console Experience

The `kill` command uses an interactive terminal UI dashboard rather than printing continuous verbose system logs:

1.  **Plan Loading Animation**: A dot loader progress bar (`0% -> 100%`) runs. Once completed, it displays:
    `✓ Plan loaded successfully. [:::::::::::::::::::::::::::::100%]`
    followed by the count of scheduled resources.
2.  **Progressive Status Updates**: Displays the currently active resource:
    `[i/N] <Type> : <Name>...... Status  : Deleting...`
3.  **In-Place Success Hiding**: Successful resource deletions are displayed briefly with a green checkmark (`✓`) and then cleared from the terminal, keeping the display neat.
4.  **Pending Status Warning**: Disabling CloudFront resources prompts a yellow warning (`⚠`) indicating pending propagation.
5.  **Persistent Failure Display**: If a deletion fails, the red cross (`✗`) error line stays persistently on the screen. For CloudFront, a custom detailed dashboard shows Status, AWS Error, Reason, and Next Action.
6.  **Developer Logging**: Verbose polling ticks, AWS retry statements, and client parameters are redirected to `reports/raw_execution.log` rather than stdout, keeping the output clean.

---

## Verification & Error Resilience

*   If a deletion step fails (e.g. a timeout or eventual consistency delay blocks a subnet from deleting), the killer records it under `failed_resources` in `reports/result.json`, leaves the failure on screen, and **continues** processing the remaining independent branches.
*   Pending resource operations are tracked under `pending_resources` in `reports/result.json`.
*   The final execution summary is printed showing the counts of deleted, pending, and failed resources. If resources are in a pending state, the status is reported as `PENDING` or `PARTIAL SUCCESS`.
