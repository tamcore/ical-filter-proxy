# AGENTS.md

Guidance for AI coding agents (Claude Code, Codex, etc.) working in this repository.

`CLAUDE.md` is a symlink to this file — keep all agent guidance here, not there.

## Repository overview

`ical-filter-proxy` is a small Go HTTP service that proxies remote iCalendar
(`.ics`) feeds and filters their events according to per-calendar rules defined
in a YAML config. It is a **drop-in replacement** for the Ruby project
[`darkphnx/ical-filter-proxy`](https://github.com/darkphnx/ical-filter-proxy):
identical HTTP surface, identical `config.yml` schema, identical filter semantics.

Single static binary, distroless image, no stateful dependencies. Upstream feeds
are fetched fresh on every request (no caching — matching upstream behavior).

## Local secrets file: AGENTS.md.local

Deployment-specific configuration (registry hosts, ingress FQDNs, kube context,
GitOps wiring) lives in **`AGENTS.md.local`** at the repo root. That file is
**gitignored** and must never be committed. If you need to deploy a dev build to
the user's cluster, read `AGENTS.md.local` for the values to use.

If `AGENTS.md.local` is missing, ask the user to populate it — do **not** infer
or hardcode private values from prior sessions, history, or context summaries.

## Privacy boundary (non-negotiable)

The following are **PRIVATE** and must never appear in tracked files (Helm
values, Makefiles, code, README, workflows, commit messages):

- Internal/private registry hostnames.
- Public IPs and ingress FQDNs that resolve to user-owned infrastructure.
- The `kube-context` name used for the user's cluster.
- The user's GitOps app name and controller.
- Anything else explicitly marked private in `AGENTS.md.local`.

If you find any of these in the working tree before committing, treat it as a
release-blocking bug and remove it. Pass them only at deploy time.

## Documentation discipline

- **`README.md` and `AGENTS.md` must be kept up-to-date at all times.** Any
  change to how a user runs, builds, deploys, or configures the app updates
  `README.md` in the same commit.

## Workflow rules

### Git
- Conventional Commits (`feat:`, `fix:`, `chore:`, `test:`, `docs:`, `refactor:`, `perf:`, `ci:`, `build:`).
- One logical change per commit; small and reviewable.

### TDD
- Write the failing test first, then the implementation. Coverage gate is
  **≥ 80 %** for `internal/...`, enforced in CI.
- Integration tests that hit the network must respect `testing.Short()` so
  `go test -short ./...` (the pre-commit hook) stays offline and fast.

### Pre-commit / CI gate
- `go vet ./...`, `go test ./...`, and `golangci-lint run` must pass before commit
  (enforced by `.pre-commit-config.yaml`).
- All CI workflows must be green before tagging a release (`v*`) or deploying.

## Project layout

```
cmd/ical-filter-proxy/   # entrypoint: flags/env, slog, embedded tzdata, HTTP server
internal/config/         # YAML schema, env substitution, startup validation
internal/filter/         # rule engine: operators, negation, regex translation
internal/calendar/       # fetch upstream .ics, apply filters + alarms, serialize
internal/server/         # HTTP handlers, auth, status codes
internal/version/        # build-time version info (ldflags)
charts/ical-filter-proxy/  # Helm chart (registry/host are caller-supplied)
```

## Config & behavior

See `README.md` for the full `config.yml` schema. Key invariants to preserve for
drop-in compatibility:

- Route `GET /<calendar_name>?key=<api_key>`; `GET /` returns a welcome string.
- `200` on success (`Content-Type: text/calendar`, CRLF line endings), `403` on
  missing/wrong key, `404` on unknown calendar, `5xx` on upstream fetch failure.
- All rules in a calendar are AND-ed; an array `val` within a rule is OR-ed.
- Operators `equals|startswith|includes|matches` with optional `not-` prefix.
- `${ICAL_FILTER_PROXY_NAME}` placeholders in the config are substituted from env
  before YAML parsing.

## Things to avoid

- Hardcoding the user's registry host, FQDN, IP, or kube context anywhere tracked.
- Adding caching or extra endpoints — keep it a faithful drop-in.
- Skipping CI before deploy "just this once".
