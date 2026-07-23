# Planning Phase Guide (`aws-kill plan`)

The `plan` command maps the topological relationships of your infrastructure and allows you to select which resource clusters you want to target for deletion.

---

## Command Definition

```bash
go run . plan
```

---

## Execution Lifecycle

The planning command goes through four distinct execution stages:

### 1. Guard Check
Verifies that `reports/inventory.json` exists. If missing, it exits with error code `1` and instructs you to run `scan` first.

### 2. Recursive Network Grouping
To separate project dependencies, the planner builds a Directed Acyclic Graph (DAG) and recursively resolves the VPC parent of every resource:
*   **VPC Groups**: Named `VPC: vpc-xxx`. Includes all subnets, route tables, custom security groups, EC2 instances, RDS databases, and ALBs associated with that VPC.
*   **Global / Independent Group**: Includes resources with no VPC linkage (S3 buckets, CloudFront distributions, or orphan key pairs).

### 3. Interactive Terminal UI (TUI) Selection
The CLI displays an interactive multi-select menu (using arrow keys and spacebar to select):
```text
? Select resource groups to target for deletion:
  [x] VPC: vpc-002b2fbb16e4f3c1e (18 resources: EC2 (1), Subnets (3)...)
  [x] Global / Independent Resources (2 resources: S3 (1), CloudFront (1))
```
This lets you target one specific network environment (Project A) while keeping another network environment (Project B) untouched.

### 4. Topological Sort & Pre-Deletion Confirmation
After selection:
1.  **Kahn's Priority Queue Algorithm**: Resolves explicit dependencies (e.g. Subnet -> VPC) and applies implicit tier priorities to sort the deletion order safely (e.g. EC2 instances must be terminated before their security groups are deleted).
2.  **Visual Plan Preview**: Displays the final planned steps sequentially.
3.  **Final Confirmation Prompt**:
    `? Are you sure you want to write this deletion plan containing 20 steps? Once written, running 'kill' will destroy them without further prompts.`
    *This is the single confirmation point in the application lifecycle.*
4.  **Save Plan**: Writes the deletion list to `reports/plan.json`.
