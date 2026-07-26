# Changelog

All notable changes to the AWS Kill Switch project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.0.0] - 2026-07-26

### Added
- **Live AWS scanning**: Resource discovery across 15 core services, automatically skipping default AWS VPCs/subnets.
- **Dependency planning**: Topological-sort dependency routing engine to schedule cleanups in safe orders, with cycle detection.
- **Interactive TUI**: Group-based network VPC checklist interface for deletion selections.
- **Sequential Kill workflow**: Sequenced active deletion loops with dynamic terminal loader bars and raw log routing.
- **Live Verification**: Direct AWS API fact-checking engine to audit deletions in real time.
- **Troubleshooter Explain command**: Developer diagnostic analyzer mapping errors (S3 non-empty, Security Group attachments) into actionable CLI fixes.
- **CloudFront Stateful Lifecycle Handling**: Automatic update configurations to disable distributions and route pending propagation state warnings cleanly.

---

## [v1.1.0] - Future Planning

### Planned
- Multi-region parallel scanning capabilities.
- Custom tagging filter exclusions inside interactive UI.
- Direct AWS cleanup action retry flags (`kill --retry`).
