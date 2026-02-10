---
name: ddcli
description: Search, aggregate, and tail Datadog logs and fetch APM traces using the ddcli CLI. Use when debugging production issues, investigating errors, triaging incidents, checking service health, inspecting traces, or when the user mentions Datadog logs, traces, log search, or error investigation. Triggers on requests involving log analysis, trace inspection, service debugging, error counts, or production monitoring.
compatibility: Requires ddcli binary (curl -sSL https://raw.githubusercontent.com/ethanrcohen/ddcli/main/install.sh | sh) and DD_API_KEY + DD_APP_KEY environment variables.
metadata:
  author: ethanrcohen
  version: "0.2.0" # x-release-please-version
---

# ddcli - Datadog CLI

**Install:** `curl -sSL https://raw.githubusercontent.com/ethanrcohen/ddcli/main/install.sh | sh`

**Requires:** `DD_API_KEY` and `DD_APP_KEY` environment variables.

## Choose Your Workflow

| Goal | Command |
|------|---------|
| Find errors in a service | [Search Logs](#search-logs) |
| Count errors / compute metrics | [Aggregate Logs](#aggregate-logs) |
| Watch logs in real time | [Tail Logs](#tail-logs) |
| Inspect a trace's spans | [Get Trace](#get-trace) |

---

## Search Logs

The primary command. Returns log entries matching a query.

```bash
# Errors in a service in the last hour
ddcli logs search --service payment --status error --from 1h

# Filter by service + environment
ddcli logs search --service user-service --env production --from 15m

# Combine flags with Datadog query syntax for advanced filters
ddcli logs search --service payment --env prod "@duration:>5s" --from 1h

# Absolute time range
ddcli logs search -s payment --from "2024-01-01T00:00:00Z" --to "2024-01-02T00:00:00Z"

# Control result count
ddcli logs search -s payment --status error --from 24h --limit 200
```

### Output Formats

| Format | Flag | Use when |
|--------|------|----------|
| JSON | `--output json` (default) | Piping to `jq`, programmatic analysis |
| Table | `--output table` | Human-readable overview |
| Raw | `--output raw` | Just log messages, one per line |

```bash
# Structured JSON (default) - pipe to jq for field selection
ddcli logs search -s payment --from 1h | jq '.data[].attributes.message'

# Human-readable table
ddcli logs search -s payment --from 1h --output table

# Just messages
ddcli logs search -s payment --from 1h --output raw
```

### Search Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--service` | `-s` | Filter by service name |
| `--env` | `-e` | Filter by environment (prod, staging) |
| `--host` | | Filter by host |
| `--status` | | Filter by log status (error, warn, info, debug) |
| `--from` | | Start time: relative (15m, 1h, 24h, 7d) or ISO 8601 |
| `--to` | | End time (default: now) |
| `--limit` | | Max results (default: 50, max: 1000) |
| `--sort` | | `-timestamp` (newest first, default) or `timestamp` |
| `--output` | `-o` | json, table, or raw |

---

## Aggregate Logs

Compute metrics from logs — counts, averages, etc. Useful for triage.

```bash
# How many errors per service in the last 24h?
ddcli logs aggregate --status error --compute count --group-by service --from 24h

# Average request duration by service
ddcli logs aggregate --compute "avg:@duration" --group-by service --from 1h

# Error count by status for one service
ddcli logs aggregate --service payment --compute count --group-by status --from 6h

# Table output for readability
ddcli logs aggregate --status error --compute count --group-by service --from 1h -o table
```

### Compute Options

| Compute | Example | Description |
|---------|---------|-------------|
| `count` | `--compute count` | Count matching logs |
| `avg:<metric>` | `--compute avg:@duration` | Average of a numeric attribute |
| `sum:<metric>` | `--compute sum:@bytes` | Sum of a numeric attribute |
| `min:<metric>` | `--compute min:@latency` | Minimum value |
| `max:<metric>` | `--compute max:@latency` | Maximum value |

---

## Tail Logs

Stream logs in real time. Polls every 2 seconds by default.

```bash
ddcli logs tail --service payment
ddcli logs tail --service payment --status error --output raw
ddcli logs tail --host web-1 --interval 5s
```

---

## Get Trace

Fetch all spans for a trace ID. Exhaustively paginates to retrieve the complete trace.

```bash
# Get all spans for a trace (JSON)
ddcli traces get <trace_id> --from 1h

# Human-readable span tree
ddcli traces get <trace_id> --from 1h --output table

# Export for visualization in Perfetto / speedscope
ddcli traces get <trace_id> --from 1h -o perfetto > trace.json
npx speedscope trace.json
```

### Trace Output Formats

| Format | Flag | Use when |
|--------|------|----------|
| JSON | `--output json` (default) | Piping to `jq`, programmatic analysis |
| Table | `--output table` | Span tree with service, resource, duration |
| Raw | `--output raw` | One span per line (compact JSON) |
| Perfetto | `--output perfetto` | Chrome Trace Event Format for visualization |

### Trace Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--from` | | Start time: relative (15m, 1h, 24h, 7d) or ISO 8601 |
| `--to` | | End time (default: now) |
| `--output` | `-o` | json, table, raw, or perfetto |

---

## Datadog Query Syntax

The `--service`, `--env`, `--host`, and `--status` flags are shortcuts that prepend to the query. You can also use raw Datadog query syntax directly:

```
service:my-service              Filter by service
status:error                    Filter by log status
host:my-host                    Filter by host
env:production                  Filter by environment
@duration:>5s                   Numeric attribute filter
"exact phrase"                  Exact match
service:web AND status:error    Boolean operators (AND, OR, NOT)
service:web-*                   Wildcards
```

## Common Investigation Patterns

```bash
# 1. Start broad: what services have errors?
ddcli logs aggregate --status error --compute count --group-by service --from 1h -o table

# 2. Drill into the top offender
ddcli logs search --service payment --status error --from 1h --output table

# 3. Get full JSON details for a specific timeframe
ddcli logs search --service payment --status error --from 30m --limit 10

# 4. Check if it's environment-specific
ddcli logs aggregate --service payment --status error --compute count --group-by env --from 1h

# 5. Tail to watch if it's ongoing
ddcli logs tail --service payment --status error

# 6. Inspect a specific trace
ddcli traces get <trace_id> --from 1h --output table

# 7. Visualize a trace
ddcli traces get <trace_id> --from 1h -o perfetto > trace.json && npx speedscope trace.json
```

## Time Ranges

`--from` and `--to` accept relative durations or absolute timestamps:

| Input | Meaning |
|-------|---------|
| `15m` | 15 minutes ago |
| `1h` | 1 hour ago |
| `24h` | 24 hours ago |
| `7d` | 7 days ago |
| `2w` | 2 weeks ago |
| `2024-01-01T00:00:00Z` | Absolute ISO 8601 |
| `now` | Current time (default for --to) |
