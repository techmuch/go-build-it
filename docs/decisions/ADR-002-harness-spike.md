# ADR-002: Harness headless and usage findings (Q2)

Status: Accepted · 2026-11-08 · Tracker: P0-04

| Harness | Headless command | Event stream | Context usage | Window size | Rate limits |
|---|---|---|---|---|---|
| Claude Code 2.1 | `claude -p <prompt> --output-format stream-json --verbose` | NDJSON: `system/init`, `assistant`, `user`, `rate_limit_event`, `result` | **Measured**: each `assistant.message.usage` (input + cache_creation + cache_read) | **Reported** in `result.modelUsage.<model>.contextWindow` | **Reported**: `rate_limit_event.rate_limit_info` with `status`, `resetsAt`, utilization per window |
| Antigravity (`agy`) | `agy -p <prompt> --output-format stream-json` | NDJSON: `init`, `step_update` (text deltas), `result` | Per step `usage.total_tokens` | **Not reported**: estimate from a per-model table | Not observed in the event stream; detect from error results |
| Pi (added P9-04) | `pi -p --mode json --no-session <prompt>` (stdin must be closed) | NDJSON: `session`, `agent_start`, `message_start/end`, `turn_end`, `agent_end`, `auto_retry_start` | Per message `usage` (input + output + cacheRead + cacheWrite) | Not reported: estimate | Detect from `errorMessage` |
| OpenCode | `opencode run --format json` (documented) | JSON events per message | Token counts per message (documented) | Not reported: estimate | Detect from error output |

OpenCode is not installed on the build machine, so its adapter is written against the documented interface and marked **unverified** until it runs once.

Fixtures from the spike live in `internal/harness/*/testdata/`. All adapters report `Usage{Used, Window, Measured}` (FR-71). Only Claude Code sets `Measured=true` for both values.

Pi's event format was recorded from a real run (`testdata/pi-connection-error.jsonl`), but its configured local model server was not running, so a successful turn has not been observed yet: the adapter is marked unverified.
