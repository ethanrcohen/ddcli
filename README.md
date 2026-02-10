# ddcli

A Datadog CLI for AI agents and humans. Provides structured access to Datadog logs from the command line with JSON, table, and raw output formats.

## Install

### From source

```bash
go install github.com/ethan/ddcli@latest
```

### Build locally

```bash
git clone https://github.com/ethan/ddcli.git
cd ddcli
make build
./ddcli --help
```

## Configuration

### Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DD_API_KEY` | Yes | Datadog API key ([create one](https://app.datadoghq.com/organization-settings/api-keys)) |
| `DD_APP_KEY` | Yes | Datadog application key ([create one](https://app.datadoghq.com/organization-settings/application-keys)) |
| `DD_SITE` | No | Datadog site (default: `datadoghq.com`). Use `datadoghq.eu` for EU, `us5.datadoghq.com` for US5, etc. |

```bash
export DD_API_KEY=your-api-key
export DD_APP_KEY=your-app-key
export DD_SITE=datadoghq.com
```

### Config file

Alternatively, use `ddcli configure` to save credentials to `~/.ddcli.json`:

```bash
ddcli configure --api-key <key> --app-key <key>
ddcli configure --api-key <key> --app-key <key> --site datadoghq.eu
```

Environment variables always take precedence over the config file.

## Usage

### Search logs

```bash
# Search with convenience flags
ddcli logs search --service payment --status error --from 1h
ddcli logs search --service web-store --env prod --from 15m

# Search with Datadog query syntax
ddcli logs search "service:payment status:error" --from 1h
ddcli logs search "@duration:>5s" --service web-store --from 15m --limit 100

# Time ranges: relative (15m, 1h, 24h, 7d) or absolute ISO 8601
ddcli logs search -s payment --from "2024-01-01T00:00:00Z" --to "2024-01-02T00:00:00Z"

# Output formats
ddcli logs search -s payment --from 1h --output json    # structured (default)
ddcli logs search -s payment --from 1h --output table   # human-readable
ddcli logs search -s payment --from 1h --output raw     # message field only
```

### Aggregate logs

```bash
# Count errors by service
ddcli logs aggregate --status error --compute count --group-by service --from 24h

# Average request duration by service
ddcli logs aggregate --compute "avg:@duration" --group-by service --from 1h

# Table output
ddcli logs aggregate --status error --compute count --group-by service --from 1h -o table
```

### Tail logs

```bash
# Stream logs in real time
ddcli logs tail --service payment
ddcli logs tail --service payment --status error --output raw
ddcli logs tail --host web-1 --interval 5s
```

### Common flags

These flags are available on all log commands (`search`, `aggregate`, `tail`):

| Flag | Short | Description |
|------|-------|-------------|
| `--service` | `-s` | Filter by service name |
| `--env` | `-e` | Filter by environment (e.g. `prod`, `staging`) |
| `--host` | | Filter by host |
| `--status` | | Filter by log status (`error`, `warn`, `info`, `debug`) |
| `--output` | `-o` | Output format: `json`, `table`, or `raw` |

These flags are syntactic sugar that get prepended to the query string, so `--service payment --status error` is equivalent to the query `service:payment status:error`.

## Development

```bash
make build          # Build the binary
make test           # Run all tests
make test-unit      # Run unit tests only
make lint           # Run linter (requires golangci-lint)
```

## Query syntax

ddcli passes queries directly to the [Datadog log search API](https://docs.datadoghq.com/logs/explorer/search_syntax/). Common patterns:

```
service:my-service              Filter by service
status:error                    Filter by status
host:my-host                    Filter by host
env:production                  Filter by environment
@duration:>5s                   Filter by custom attribute
"exact phrase"                  Match exact phrase
service:web AND status:error    Boolean operators (AND, OR, NOT)
service:web-*                   Wildcards
```
