# API Dual-Environment Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one Mac-side command to deploy the Go API to isolated test or production services on the existing Alibaba ECS.

**Architecture:** A Bash CLI builds Linux amd64 from either the current worktree or a clean `origin/main` archive, uploads a versioned binary over SSH, and installs/restarts only the selected systemd service. Nginx and WeChat uploads remain manual. Production has a fail-closed runtime check against demo authentication.

**Tech Stack:** Bash, Go 1.24.3, Git, SSH/SCP, systemd, curl.

**Spec:** `docs/superpowers/specs/2026-11-17-api-deployment-design.md`

## Global Constraints

- ECS target: `root@39.105.122.224`, SSH port 22, Linux amd64.
- `test`: local working tree, port 8082, separate data, default `APP_ENV=local-demo`.
- `production`: fetched `origin/main`, port 8081, `APP_ENV=production`, separate MySQL configuration.
- Do not mutate Nginx, certs, miniapp or admin-web.
- Do not overwrite the current user modifications in the repository.

## Review Focus

- An invalid or mismatched IP:port must fail before compiling or contacting ECS.
- Failed tests/build/SSH must leave both active services untouched.
- A production binary that still exposes demo authentication must be rejected and rolled back.
- A failed restart must restore the selected environment's previous release only.
- Production secrets must never appear in script output or Git.

---

### Task 1: CLI and source selection

**Files:** Create `scripts/deploy-api.sh`; create `scripts/test-deploy-api.sh`.

**Interfaces:** `deploy-api.sh --mode test|production --address HOST[:PORT] [--dry-run]`.

- [ ] Write shell tests for argument validation, test worktree source, production `origin/main` source, and no remote calls on invalid input.
- [ ] Run tests and observe failures.
- [ ] Implement parsing, URL normalization, source preparation, Go test/build, and temp cleanup.
- [ ] Run tests and `bash -n scripts/deploy-api.sh`.

### Task 2: Remote install and rollback

**Files:** Modify `scripts/deploy-api.sh`; extend `scripts/test-deploy-api.sh`.

**Interfaces:** Upload binary with SCP; run a remote Bash installer over SSH with mode, release ID, and expected port as arguments.

- [ ] Write shell tests for remote command targeting, separate service names and ports, and failure paths.
- [ ] Run tests and observe failures.
- [ ] Implement remote preflight, isolated env/data directories, versioned releases, systemd unit, restart, local HTTP check, production demo-route guard, and rollback.
- [ ] Run shell tests and syntax check.

### Task 3: Operator instructions and full verification

**Files:** Create `docs/deployment-api.md`. Existing `README.md` has concurrent user edits, so leave it untouched.

- [ ] Document one-time SSH, security-group and production env prerequisites; show exact test/production invocations and expected results.
- [ ] Run shell tests, Go tests, shell syntax check, and script help/dry-run.
- [ ] Review the diff for unintended source changes and secrets.
