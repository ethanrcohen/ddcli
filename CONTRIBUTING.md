# Contributing to ddcli

## Architecture Decisions

### Why Go + Cobra (not TypeScript)

We chose Go over TypeScript despite the team being primarily TS developers because:

- **Single binary distribution** — no runtime dependency. Teammates `curl | sh` or `brew install`, no Node/npm/npx needed on the target machine.
- **Cobra** is the standard CLI framework in Go (used by `kubectl`, `gh`, `docker`). Excellent subcommands, auto-generated help, shell completions.
- **Cross-compilation** is trivial — `GOOS=linux GOARCH=arm64 go build` just works. GoReleaser builds all 4 binaries (macOS/Linux x amd64/arm64) in seconds.
- The team *uses* the CLI, they don't develop it. The install experience matters more than the source language.

### Why raw HTTP client (not the Datadog Go SDK)

We use `net/http` directly against the Datadog API v2 instead of the official `datadog-api-client-go` SDK because:

- **Lighter dependency tree** — the SDK pulls in a large number of transitive dependencies.
- **Full control** over request/response handling, error parsing, and pagination.
- **Testability** — the `api.Client` accepts a `BaseURL` parameter, so we can point it at `httptest` servers in tests. The `LogsAPI` interface enables mock-based unit testing of commands.

### Why JSON output by default

The primary audience is AI agents. JSON is structured, parseable, and composable with tools like `jq`. Agents can extract exactly the fields they need. Table and raw formats exist for human use.

### Convenience flags (`--service`, `--env`, etc.)

These flags exist on all log commands (search, aggregate, tail) as syntactic sugar. They prepend to the Datadog query string via `buildQuery()` in `cmd/logs_query.go`. This makes the most common filters discoverable via `--help` without requiring agents or users to know Datadog query syntax. You can mix flags with raw query syntax freely:

```bash
ddcli logs search --service payment --env prod "@duration:>5s"
# becomes query: "service:payment env:prod @duration:>5s"
```

## Datadog API Endpoints

The CLI talks to these Datadog API v2 endpoints:

| Command | Endpoint | Notes |
|---------|----------|-------|
| `logs search` | `POST /api/v2/logs/events/search` | Cursor-based pagination, max 1000 per page |
| `logs aggregate` | `POST /api/v2/logs/analytics/aggregate` | Computes metrics over log data |
| `logs tail` | `POST /api/v2/logs/events/search` (polling) | Not true streaming — polls every 2s |

Authentication is via `DD-API-KEY` and `DD-APPLICATION-KEY` headers on every request.

### Rate limits

Datadog returns HTTP 429 when rate limited. Currently we surface the error but **do not retry with backoff**. This is a known gap.

## Project Structure

```
cmd/                       # Cobra command definitions
  root.go                  # Entry point, top-level help
  configure.go             # ddcli configure
  logs.go                  # ddcli logs (parent command)
  logs_query.go            # Shared: addLogFilterFlags(), buildQuery()
  logs_search.go           # ddcli logs search
  logs_aggregate.go        # ddcli logs aggregate
  logs_tail.go             # ddcli logs tail
  logs_search_test.go      # Command-level tests with mock API
  logs_aggregate_test.go
  logs_query_test.go       # Unit tests for buildQuery()
internal/
  api/
    client.go              # HTTP client, LogsAPI interface, error types
    logs.go                # SearchLogs, AggregateLogs implementations + request/response types
    client_test.go         # httptest-based API client tests
  config/
    config.go              # Env var + file config, precedence logic
  output/
    formatter.go           # Format enum, parser, factory functions
    json.go / table.go / raw.go
    formatter_test.go      # Output format tests
  timeutil/
    parse.go               # Relative duration + ISO 8601 parsing
    parse_test.go          # Table-driven time parsing tests
skill/
  SKILL.md                 # Agent skill (agentskills.io spec)
install.sh                 # One-line installer for binary releases
```

## Testing

### Strategy

Tests are layered by what they cover:

| Layer | Technique | What it tests | Files |
|-------|-----------|---------------|-------|
| Business logic | Table-driven unit tests | Time parsing, query building, compute parsing, output formatting | `*_test.go` in `internal/`, `cmd/logs_query_test.go` |
| API client | `net/http/httptest` servers | HTTP request construction, headers, serialization, error handling | `internal/api/client_test.go` |
| Commands | Interface mock (`mockLogsAPI`) | Full command flow: flag parsing → API call → output formatting | `cmd/logs_search_test.go`, `cmd/logs_aggregate_test.go` |

### Running tests

```bash
make test           # All tests, verbose
make test-unit      # Unit tests only (-short flag)
go test ./... -v    # Same as make test
```

### Writing new tests

**For new commands:** Inject the mock API via the `SetLogs*Deps()` pattern (see `cmd/logs_search.go`). This injects both a mock `LogsAPI` and a fixed `time.Now()` for deterministic tests.

**For API client changes:** Use `httptest.NewServer` to verify the exact HTTP request (method, path, headers, body) and test error responses (403, 429, etc.). See `internal/api/client_test.go`.

**Cobra rootCmd caveat:** Cobra's `rootCmd` is shared across tests in the same package. Flag values set in one test leak to the next. When a test sets flags like `--service` or `--host`, subsequent tests must explicitly reset them to `""` or they'll carry over. Example:

```go
rootCmd.SetArgs([]string{
    "logs", "search", "@duration:>5s",
    "--service", "payment",
    "--host", "",       // reset from previous test
    "--status", "",     // reset from previous test
    "--from", "1h",
})
```

### Not yet implemented

These testing approaches were designed but not yet built:

- **Golden file / snapshot tests** — run the CLI binary, capture stdout, compare against `.golden` files. Catches output format regressions.
- **Record/replay (go-vcr)** — record real Datadog API responses, scrub secrets, replay in CI for high-fidelity tests without credentials.
- **Live integration tests** — gated behind `//go:build integration`, hit real Datadog API. The `Makefile` has a `test-integration` target but no integration tests exist yet.

## Releasing

### Prerequisites

- [GoReleaser](https://goreleaser.com/): `brew install goreleaser`
- GitHub CLI: `brew install gh` (already authenticated)

### Cut a release

```bash
# 1. Tag the release
git tag v0.2.0

# 2. Push the tag
git push origin main --tags

# 3. Build binaries and create GitHub release
GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
```

This builds 4 binaries (macOS/Linux x amd64/arm64), creates a GitHub release with changelog, and uploads the archives.

### Homebrew tap

The `.goreleaser.yaml` does not currently include a Homebrew tap configuration. To enable `brew install ethanrcohen/tap/ddcli`, you'd need to:

1. Create a `ethanrcohen/homebrew-tap` repo on GitHub
2. Add the `brews` section back to `.goreleaser.yaml`:
   ```yaml
   brews:
     - repository:
         owner: ethanrcohen
         name: homebrew-tap
       name: ddcli
       homepage: "https://github.com/ethanrcohen/ddcli"
       description: "Datadog CLI for AI agents and humans"
   ```

## Known Gaps / Future Work

- **Retry with backoff on 429** — currently rate limit errors surface directly to the user
- **Timeseries aggregation** — `--type timeseries --interval 5m` was designed but not implemented
- **`--fields` flag** — discussed and deliberately deferred; agents can pipe JSON to `jq`
- **`logs tail` is polling-based** — Datadog doesn't offer a streaming API, so we poll every 2s
- **No Windows builds** — GoReleaser config only builds darwin/linux. Add `windows` to `goos` if needed
