# ADR-007: When smart handoffs graduate from experimental (Q3)

Status: Accepted · approved by the owner 2026-11-08 · Tracker: P9-02

Smart handoffs stay experimental until `gbi metrics compare` shows at least 10 handed-off tasks in the last 30 days, and their rework rate (reviews requesting changes per task) is no worse than compaction-only tasks. Then the owner holds a graduation review. If it passes, the default threshold becomes 75% (Q3), configurable per harness through `experimental.handoff_threshold` until per-harness overrides are needed.
