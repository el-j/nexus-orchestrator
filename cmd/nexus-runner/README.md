# nexus-runner

`nexus-runner` is a standalone CLI for repository verification gates, clean architecture policy audits, and automated self-healing task execution loops.

It requires no daemon or background database, making it ideal for CI/CD pipelines, pre-commit hooks, and terminal workflows.

---

## Commands

### 1. Verification Gate (`verify`)

Executes a command and evaluates both the command's exit code and architectural policy rules:

```sh
# Run tests with verification checks
nexus-runner verify --command "go test -v ./..."

# Specify a custom policy file
nexus-runner verify --command "npm test" --policy .nexus/policy.json
```

### 2. Self-Healing Execution (`execute`)

Executes a command iteratively. When failures occur, it captures compiler/test errors, queries an available LLM provider for a patch, applies extracted code modifications, and re-tests:

```sh
nexus-runner execute \
  --command "go test -v ./internal/core/..." \
  --max-attempts 3 \
  --prompt "Fix failing unit tests in core services"
```

### 3. Invariant Policy Audit (`inspect`)

Scans the repository for clean architecture constraints:

- Source file line limits ($\le 500$ lines).
- Test file line limits ($\le 800$ lines).
- Test-first conventions.
- Git working tree cleanliness.

```sh
# Inspect current directory
nexus-runner inspect

# Inspect a specific sub-package
nexus-runner inspect --dir ./pkg/runner
```
