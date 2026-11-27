# AGENTS.md

Guidance for AI coding agents (and humans) working in this repository.

## What this is

stefunny is a CLI that deploys AWS Step Functions state machines plus the EventBridge rules,
EventBridge Scheduler schedules and aliases that come with them. User-facing docs are in `README.md`.

## Layout

- Root package `stefunny` holds almost all logic: CLI definitions (`cli.go`, kong), config loading and
  templating (`config.go`, jsonnet/YAML), commands (`deploy.go`, `diff.go`, `render.go`, `rollback.go`, ...),
  and thin service layers over the AWS SDK (`sfn_service.go`, `eventbridge_service.go`, `scheduler_service.go`).
- `cmd/stefunny/` is the `main` package only.
- `internal/sfnx`, `internal/eventbridgex`: paginators for AWS APIs the SDK does not cover.
- `mock/`: **generated** gomock mocks. Do not edit by hand.
- `testdata/`: fixtures and golden files (`*.golden.*`).

## Commands

Tasks are defined in `Taskfile.yml` ([go-task](https://taskfile.dev)). Install with
`go install github.com/go-task/task/v3/cmd/task@latest`. Run `task` to list them.

| Task | What it does |
|---|---|
| `task test` | `go test -race -timeout 30s ./...` (same as CI). Pass flags after `--`: `task test -- -run TestRender` |
| `task lint` | golangci-lint at the version CI uses, with `.golangci.yaml` |
| `task generate` | `go generate ./...` (regenerates `mock/`) |
| `task test:update-golden` | `go test . -update` (rewrites golden files in `testdata/`) |
| `task vuln` | govulncheck |
| `task check` | Everything CI checks. Run this before you say a change is done |

Unit tests need no AWS account or network: AWS clients are interfaces and tests use the mocks in `mock/`.
`docker-compose.yaml` and the `Makefile` (localstack, Step Functions Local, terraform) are only for manual
local runs and refreshing `testdata/terraform.tfstate`; see `docs/local_deelopment.ja.md`.

## Conventions

- Go version comes from `go.mod`. Do not lower it.
- When you change an interface that has a `//go:generate ... mockgen` line, run `task generate` and commit
  the result. CI fails if `mock/`, `go.mod` or `go.sum` are out of date.
- When an output format changes on purpose, run `task test:update-golden`, then read the golden diff and
  make sure every change is intended. Never update golden files just to make a failing test pass.
- The state machine definition is a user document: do not prune or normalize it (see issue #351).
  Zero-value pruning (`deleteNilFromMap` in `encoding.go`) is only for AWS SDK-derived config values.
- Keep behavior changes covered by tests in the matching `*_test.go` file.
- Commit messages follow Conventional Commits with a scope, e.g. `fix(render): ...`, `feat(diff): ...`.

## Release (do not touch)

Releases are driven by [tagpr](https://github.com/Songmu/tagpr) and GoReleaser and are done by the maintainer.
Do not edit `version.go`, `CHANGELOG.md` or `CREDITS` by hand, and do not merge the tagpr release PR.
