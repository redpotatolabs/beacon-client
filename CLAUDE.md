# beacon-client

Public Go module with the client and wire types for beacon's HTTP API
(`/v1/errors`, `/v1/deploys`). Beacon itself (`redpotatolabs/beacon`) and the
libraries that report to it, such as discord-bot-lib, pin this module.

The Red Potato Labs standing rules come from the flow plugin. This file only
holds what is specific to beacon-client.

## Workflow

- A library: it deploys nothing, so there is no test site and no `test`
  branch.
- Work goes on a branch from `main` and straight to a pull request to `main`.
  `main` takes only squash-merged pull requests with a passing `checks` job.
  Never push to `main` and never force push.
- Before pushing, run `go vet ./...` and `go test -race ./...`.
- CI (`.github/workflows/ci.yml`) runs on a GitHub-hosted runner with a
  read-only token and no secrets, because the repo is public and anyone can
  open a pull request from a fork. Never move it to the self-hosted runners
  and never add a secret or a `redpotatolabs/actions` step to it.
- A change to the wire types is a change to beacon's API. There are no tags
  yet: callers pin a commit (pseudo-version), so bump it in beacon and the
  libraries that use it after the merge.

## Facts

- Linear: project P-RED-3 "Productionize", Initiative label
  `Infrastructure`.
- No RedQueen notes: this is internal.
