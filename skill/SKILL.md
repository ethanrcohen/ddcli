---
name: datadog-observability
description: Search, aggregate, and analyze Datadog logs, metrics, and APM data using pup (Datadog's official CLI). Use when debugging production issues, investigating errors, triaging incidents, checking service health, querying metrics, or when the user mentions Datadog, logs, metrics, APM, or error investigation. Triggers on requests involving log analysis, metric queries, service debugging, error counts, or production monitoring.
compatibility: Requires pup CLI (brew install datadog/pack/pup) authenticated via `pup auth login` (OAuth2, preferred) or DD_API_KEY + DD_APP_KEY environment variables.
metadata:
  author: ethanrcohen
  version: "1.0.0"
---

# Datadog Observability Skill (via pup)

**Requires:** `pup` CLI, authenticated via `pup auth login` or `DD_API_KEY` + `DD_APP_KEY` env vars.

## Choose Your Workflow

| Goal | Command |
|------|---------|
| Find errors in a service | [Search Logs](#search-logs) |
| Count errors / compute metrics | [Aggregate Logs](#aggregate-logs) |
| Query time-series metrics | [Query Metrics](#query-metrics) |
| List APM services + perf stats | [APM Services](#apm-services) |
| View service dependencies | [APM Dependencies](#apm-dependencies) |

---

## Search Logs

Returns log entries matching a Datadog query.

```bash
# Errors in a service in the last hour
pup logs search --query="service:payment AND status:error" --from="1h"

# Filter by service + environment
pup logs search --query="service:user-service AND env:production" --from="15m"

# Advanced attribute filters
pup logs search --query="service:payment AND @duration:>5s" --from="1h"

# Control result count
pup logs search --query="service:payment AND status:error" --from="24h" --limit=200

# Sort oldest first
pup logs search --query="status:error" --from="1h" --sort="asc"
```

### Search Flags

| Flag | Description |
|------|-------------|
| `--query` | Datadog query string (required) |
| `--from` | Start time: relative (`1h`, `30m`, `7d`) or Unix ms (required) |
| `--to` | End time (default: `now`) |
| `--limit` | Max results (default: 50, max: 1000) |
| `--sort` | `asc` or `desc` (default: desc) |
| `--index` | Comma-separated log indexes |
| `--output` / `-o` | `json` (default), `table`, `yaml` |

---

## Aggregate Logs

Compute metrics from logs -- counts, averages, percentiles. Useful for triage.

```bash
# How many errors per service in the last 24h?
pup logs aggregate --query="status:error" --from="24h" --compute="count" --group-by="service"

# Average request duration by service
pup logs aggregate --query="*" --from="1h" --compute="avg(@duration)" --group-by="service"

# 99th percentile latency
pup logs aggregate --query="service:api" --from="2h" --compute="percentile(@duration, 99)"

# Error count by HTTP status code
pup logs aggregate --query="status:error" --from="1d" --compute="count" --group-by="@http.status_code"
```

### Compute Options

| Compute | Example | Description |
|---------|---------|-------------|
| `count` | `--compute="count"` | Count matching logs |
| `avg(metric)` | `--compute="avg(@duration)"` | Average of a numeric attribute |
| `sum(metric)` | `--compute="sum(@bytes)"` | Sum |
| `min(metric)` | `--compute="min(@latency)"` | Minimum |
| `max(metric)` | `--compute="max(@latency)"` | Maximum |
| `cardinality(field)` | `--compute="cardinality(@user.id)"` | Unique values |
| `percentile(metric, N)` | `--compute="percentile(@duration, 99)"` | Percentile |

---

## Query Metrics

Query time-series metrics data.

```bash
# CPU usage across all hosts in the last hour
pup metrics query --query="avg:system.cpu.user{*}" --from="1h"

# Memory for a specific service in production
pup metrics query --query="avg:system.mem.used{service:web,env:prod}" --from="4h"

# Search for available metrics
pup metrics list --filter="system.cpu.*"

# Get metadata for a specific metric
pup metrics get system.cpu.user
```

### Metrics Flags

| Flag | Description |
|------|-------------|
| `--query` | Datadog metrics query (required) |
| `--from` | Start time: relative (`1h`, `30m`, `7d`) or Unix ms (required) |
| `--to` | End time (default: now) |
| `--output` / `-o` | `json` (default), `table`, `yaml` |

---

## APM Services

List services and their performance statistics. Note: APM commands use **Unix timestamps** (not relative time).

```bash
# List all APM services
pup apm services list

# Service performance stats (last hour)
pup apm services stats --start=$(date -v-1H +%s) --end=$(date +%s)

# Filter by environment
pup apm services stats --start=$(date -v-1H +%s) --end=$(date +%s) --env=prod

# List operations for a service
pup apm services operations web-server --start=$(date -v-1H +%s) --end=$(date +%s)

# List resources (endpoints) for a service operation
pup apm services resources web-server --operation="GET /api/users" --from=$(date -v-1H +%s) --to=$(date +%s)
```

### APM Flags

| Flag | Description |
|------|-------------|
| `--start` | Start time as Unix timestamp (required for stats/operations) |
| `--end` | End time as Unix timestamp (required for stats/operations) |
| `--env` | Filter by environment |
| `--primary-tag` | Filter by primary tag (`group:value`) |
| `--output` / `-o` | `json` (default), `table`, `yaml` |

---

## APM Dependencies

View service call relationships based on trace data.

```bash
# All service dependencies in production
pup apm dependencies list --env=prod --start=$(date -v-1H +%s) --end=$(date +%s)

# Dependencies for a specific service
pup apm dependencies list web-server --env=prod --start=$(date -v-1H +%s) --end=$(date +%s)

# Service flow map with performance metrics
pup apm flow-map --query="env:prod" --from=$(date -v-1H +%s) --to=$(date +%s)
```

---

## Datadog Query Syntax

All query filters use Datadog's standard search syntax:

```
service:my-service              Filter by service
status:error                    Filter by log status
host:my-host                    Filter by host
env:production                  Filter by environment
@duration:>5s                   Numeric attribute filter
"exact phrase"                  Exact match
service:web AND status:error    Boolean operators (AND, OR, NOT)
service:web-*                   Wildcards
-status:info                    Negation
```

---

## Output Formats

All commands support `--output` / `-o`:

| Format | Flag | Use when |
|--------|------|----------|
| JSON | `--output json` (default) | Piping to `jq`, programmatic analysis |
| Table | `--output table` | Human-readable overview |
| YAML | `--output yaml` | Configuration-style output |

```bash
# Pipe JSON to jq for field selection
pup logs search --query="status:error" --from="1h" | jq '.data[].attributes.message'

# Human-readable table
pup logs search --query="status:error" --from="1h" --output table
```

---

## Common Investigation Patterns

```bash
# 1. Start broad: what services have errors?
pup logs aggregate --query="status:error" --from="1h" --compute="count" --group-by="service"

# 2. Drill into the top offender
pup logs search --query="service:payment AND status:error" --from="1h" --output table

# 3. Get full JSON details for a specific timeframe
pup logs search --query="service:payment AND status:error" --from="30m" --limit=10

# 4. Check if it's environment-specific
pup logs aggregate --query="service:payment AND status:error" --from="1h" --compute="count" --group-by="env"

# 5. Check APM service health
pup apm services stats --start=$(date -v-1H +%s) --end=$(date +%s) --env=prod

# 6. View service dependencies
pup apm dependencies list payment --env=prod --start=$(date -v-1H +%s) --end=$(date +%s)

# 7. Check a specific metric
pup metrics query --query="avg:trace.servlet.request.duration{service:payment}" --from="1h"
```

## Time Ranges

**Logs & Metrics** accept relative durations:

| Input | Meaning |
|-------|---------|
| `1h` | 1 hour ago |
| `30m` | 30 minutes ago |
| `7d` | 7 days ago |
| `1w` | 1 week ago |
| `now` | Current time (default for --to) |

**APM commands** require Unix timestamps. Use `date` to compute them:

| Shell | 1 hour ago | Now |
|-------|-----------|-----|
| macOS | `$(date -v-1H +%s)` | `$(date +%s)` |
| Linux | `$(date -d '1 hour ago' +%s)` | `$(date +%s)` |
