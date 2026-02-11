# datadog-agent-skill

An [agent skill](https://agentskills.io) that teaches AI coding agents how to query Datadog observability data using [pup](https://github.com/DataDog/pup) (Datadog's official CLI).

## What it does

When installed, this skill gives your AI agent the ability to:

- **Search logs** -- find errors, filter by service/env/status, control time ranges
- **Aggregate logs** -- count errors, compute averages, group by dimensions
- **Query metrics** -- time-series data, CPU/memory/custom metrics
- **Inspect APM services** -- performance stats, operations, endpoints
- **View dependencies** -- service call graphs, flow maps

All commands use `pup`, Datadog's official CLI, under the hood.

## Install the skill

```bash
npx skills add ethanrcohen/datadog-agent-skill
```

## Prerequisites

1. Install pup: `brew install datadog/pack/pup`
2. Authenticate: `pup auth login` (OAuth2, preferred) or set `DD_API_KEY` + `DD_APP_KEY` env vars

## Previously

This repo was `ddcli`, a standalone Datadog CLI. The CLI functionality is now covered by [DataDog/pup](https://github.com/DataDog/pup). This repo now focuses solely on the agent skill layer.
