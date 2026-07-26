# Upcoming Features & Enhancements

This document details the design specifications for five upcoming features planned for future releases of `aws-kill`. These enhancements focus on improving developer experience, system safety, and terminal visibility.

---

## 1. Visual Dependency Graph Exporter (`plan --graph`)

### Objective
Provide developers with a visual representation of their planned deletion sequence, highlighting parent-child relationships and dependency chains before destruction.

### User Experience (UX)
Running the planning phase with the graph flag:
```bash
aws-kill plan --graph
```
Generates a markdown file `reports/graph.md` containing a Mermaid.js diagram representing the topological tree.

### Technical Design
1.  **Orchestration**: During the planning topological sort in `engine/planner.go`, traverse the adjacency list of the constructed Directed Acyclic Graph (DAG).
2.  **Mermaid Mapping**: Map the edges of the DAG into Mermaid flowcharts syntax:
    ```mermaid
    flowchart TD
      VPC["VPC: vpc-12345"] --> Subnet["Subnet: subnet-abc"]
      Subnet --> ENI["Network Interface: eni-0123"]
      SecurityGroup["Security Group: sg-890"] --> ENI
      ENI --> EC2["EC2 Instance: i-999"]
    ```
3.  **Output**: Save the syntax directly inside `reports/graph.md` so it can be previewed natively in GitHub, VS Code, or other markdown viewers.

---

## 2. Live Scan Spinner (Interactive Scanner Feedback)

### Objective
Enhance terminal UI feedback during the scanning phase to indicate activity and eliminate the appearance of CLI freezing during network calls.

### User Experience (UX)
When running the `scan` command, instead of static console outputs, the CLI displays a dynamic spinner:
```text
⠋ [1/15] Scanning EC2 Instances...
⠙ [2/15] Scanning RDS Databases...
```

### Technical Design
1.  **Spinner Engine**: Integrate a lightweight CLI spinner framework or write a custom channel-based character rotator (`\`, `|`, `/`, `-`) in `cmd/scan.go`.
2.  **Progress Tracking**: Map the 15 scanning services. As each service begins execution, update the spinner prefix text.
3.  **Access Warning Handling**: If a scanner hits an AccessDenied exception, clear the spinner line in-place, log a brief warning, and resume the spinner for the next service.

---

## 3. Safety Protection Tags (`aws-kill:protected`)

### Objective
Prevent the accidental deletion of production workloads or critical sandbox resources by introducing a tagging protection fuse.

### User Experience (UX)
If the scanner encounters any resource tagged with a protection identifier, it logs it as protected:
```text
⚠ Skip Protected Resource: RDS Database 'prod-db' (Reason: tag 'aws-kill:protected=true' detected)
```

### Technical Design
1.  **Scoping Rules**: Update `engine/scanner.go` to inspect the `Tags` slice of each resource.
2.  **Match Condition**: Skip the resource if it contains either:
    *   Tag Key: `aws-kill:protected`, Value: `true`
    *   Tag Key: `Environment`, Value: `production` or `prod`
3.  **State Protection**: Exclude these resources from `reports/inventory.json` entirely. They can never be planned or killed by the utility.

---

## 4. AWS Profile Selector Prompt

### Objective
Remove setup friction by offering an interactive profile configuration selection on startup.

### User Experience (UX)
If `aws-kill scan` is executed without specifying a `--profile` flag and no default AWS credentials are set in the environment:
```text
? No AWS profile specified. Select a profile to authenticate:
  ▸ dev-sandbox
    staging-account
    personal-sandbox
```

### Technical Design
1.  **Ini Parser**: Read and parse the local AWS credentials file located at `~/.aws/credentials` or `~/.aws/config`.
2.  **User Prompt**: If multiple profiles exist, trigger a `survey.Select` dropdown prompt listing the parsed profile names.
3.  **Session Bind**: Initialize the standard `aws.Config` session using the selected profile credentials.

---

## 5. Deletion Plan Diff (Terraform Style)

### Objective
Enhance plan summaries by displaying deletion plans in a color-coded structural diff layout, mimicking standard infrastructure-as-code CLI outputs.

### User Experience (UX)
Running `aws-kill kill --dry-run` or viewing the planned preview outputs:
```diff
AWS Deletion Plan Preview:

- [DESTROY]  EC2 Instance                 : doc-on-call (i-06d9a26372d82bb8)
- [DESTROY]  Application Load Balancer    : docco-alb (arn:aws:elasticload...)
~ [DISABLE]  CloudFront Distribution      : df4zuk2qiajjq.cloudfront.net
- [DESTROY]  S3 Bucket                    : docco-frontend-prod
```

### Technical Design
1.  **Color Codes**: Utilize ANSI color escape sequences:
    *   `- [DESTROY]` styled in bright red (`\033[31m`).
    *   `~ [DISABLE]` styled in yellow (`\033[33m`).
2.  **Output Layout**: Standardize line widths and alignment for service types, IDs, and resource names.
