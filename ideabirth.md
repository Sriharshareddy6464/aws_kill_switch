# Why I Built AWS Kill Switch

This project wasn't born from trying to build the next big open-source AWS tool.

It came from a repetitive problem I kept facing while learning AWS.

Like many students and self-taught developers, I learned AWS by building projects manually through the AWS Console before gradually migrating them to Infrastructure as Code using Terraform.

When I start learning a new AWS service, I intentionally use the Console first. It helps me understand what resources AWS actually creates, how services are connected, and what Terraform is automating behind the scenes. Once I understand the architecture, I recreate it using Terraform.

This became my learning workflow:

```
Build → Learn → Document → Destroy → Repeat
```

After every project, I document the architecture, deployment process, and lessons learned on my portfolio before moving on to the next project.

The problem always started at the last step.

Deleting an AWS application is rarely as simple as pressing one button.

A load balancer depends on target groups.
Target groups depend on EC2 instances.
Subnets belong to VPCs.
Internet Gateways must be detached.
Security Groups cannot disappear while something still references them.

I found myself spending more time cleaning infrastructure than actually building new projects.

I wasn't running a production company or managing enterprise cloud infrastructure.

I was simply a student paying attention to AWS Free Tier credits and trying to avoid unnecessary cloud costs.

I tried existing cleanup tools, including AWS Nuke. They are incredibly powerful projects designed for large-scale cloud cleanup, but they also target much broader use cases. For my personal learning workflow, I wanted something smaller, easier to understand, and something I could build myself while learning Go.

At the same time, I had started learning Go.

Instead of writing another tutorial application, I decided to solve a real problem I encountered every week.

That decision became AWS Kill Switch.

The goal of this project isn't to replace mature infrastructure management tools.

Its goal is much simpler:

> Help developers discover, understand, plan, safely remove, and verify manually created AWS infrastructure.

The application follows a transparent workflow instead of immediately deleting resources.

```
Scan
    ↓
Plan
    ↓
Kill
    ↓
Verify
    ↓
Explain
```

Every phase generates reports that show exactly what the application discovered and what happened during execution.

This repository represents the first public release of that idea.

If you're a student, someone learning AWS, or a developer who manually creates cloud infrastructure during experimentation and gets tired of deleting resources one by one, I hope this project saves you some time.

## Disclaimer

AWS Kill Switch is an educational project created for development and learning environments.

It has **NOT** been designed, audited, or tested for production infrastructure.

I do **NOT** recommend using it against business-critical AWS accounts or production environments.

Deleting cloud infrastructure is irreversible.

Always review the generated execution plan before running the Kill phase.

You are fully responsible for understanding the resources being deleted and the consequences of running this application.

Use it at your own risk.
