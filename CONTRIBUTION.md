# Contributing to AWS Kill Switch

Thank you for your interest in contributing to AWS Kill Switch! We welcome community contributions to make cloud cleanups safer and more efficient.

---

## Development Setup

To set up a local development environment, make sure you have Go installed, then run:

```bash
# Clone the repository
git clone https://github.com/Sriharshareddy6464/aws-kill.git
cd aws-kill

# Install dependencies and tidy modules
go mod tidy

# Run the local build entrypoint
go run .
```

---

## Workflow Guidelines

To ensure stable integrations and clean codebase maintenance, please follow these guidelines:

1. **Focus**: Keep pull requests focused on a single feature or bug fix.
2. **Size**: Smaller, atomic pull requests are preferred and will be reviewed much faster.
3. **Tests**: Add unit tests for engine or scheduling logic changes and run verification tests:
   ```bash
   go test -v ./engine/...
   ```

---

## Commit Message Conventions

We follow structured commit message conventions to generate clean changelogs. Prefix your commits with one of the following formats:

*   `feat(<component>):` for new features (e.g. `feat(scan): support ECS clusters`)
*   `fix(<component>):` for bug fixes (e.g. `fix(rds): handle final snapshot deletion`)
*   `docs:` for documentation updates (e.g. `docs: improve README`)
*   `refactor:` for code refactoring with no behavior changes (e.g. `refactor(verify): consolidate live checks`)
*   `test:` for adding or modifying tests (e.g. `test(killer): add waiter tests`)

---

## Reporting Issues

If you discover a bug or have a feature suggestion, please open a GitHub Issue before submitting large changes. This allows the maintainers to discuss the design and prevent duplicate effort.
