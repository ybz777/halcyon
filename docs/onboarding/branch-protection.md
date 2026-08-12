# Branch Protection Configuration

This document describes the required GitHub branch protection rules for `main`.
Apply these in **Settings → Branches → Branch protection rules**.

## Why This Exists

Branch protection ensures every merge to `main` passes the Quality Gate defined in
[CONTRIBUTING.md](../../CONTRIBUTING.md). This is required before the repository
scales to multiple contributors.

## Required Settings for `main`

| Setting | Value |
|---|---|
| Require a pull request before merging | Enabled |
| Required approvals | 1 |
| Dismiss stale pull request approvals when new commits are pushed | Enabled |
| Require status checks to pass before merging | Enabled |
| Require branches to be up to date before merging | Enabled |
| Require conversation resolution before merging | Enabled |
| Do not allow bypassing the above settings | Enabled (admins included) |

## Required Status Checks

Add these checks from the CI workflow:

| Check name | Workflow | Job |
|---|---|---|
| `Format, Tidy, Vet` | CI | `quick-checks` |
| `golangci-lint` | CI | `lint` |
| `Unit Tests` | CI | `test` |
| `Build` | CI | `build` |
| `Docker Build` | CI | `docker` |
| `Conventional PR Title` | PR Checks | `conventional-title` |

After the first CI run on `main`, status check names appear in the branch
protection UI. Select all six checks above.

## Merge Settings

| Setting | Value |
|---|---|
| Allow squash merging | Enabled (default) |
| Allow merge commits | Disabled |
| Allow rebase merging | Disabled |

Squash merge keeps `main` history linear and aligns with [CONTRIBUTING.md](../../CONTRIBUTING.md).

## Related Documents

- [CONTRIBUTING.md](../../CONTRIBUTING.md) — Quality Gate
- [docs/onboarding/ci.md](ci.md) — Local CI reproduction
- [.github/workflows/ci.yml](../../.github/workflows/ci.yml)
