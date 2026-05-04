# Multi-Flow Contract

## When to Load

Load when SKILL.md first user-gated detects a request contains multiple independent flows.

## Detecting Multi-Flow

Multi-flow occurs when a request contains multiple actions that don't belong to the same flow (e.g., "create plan for A and maintain docs B").

Signals:
- Conjunctions: "and", "simultaneously", "at the same time"
- Multiple unrelated doc targets
- Multiple different modes requested

## Handling

1. **List all detected flows** clearly. Confirm with user.
2. **Check relationships between flows**:
   - Related → run sequentially (dependent flow first)
   - Unrelated → run in parallel (via subagents if supported)
3. **Switch to plan mode** if CLI/model supports it.
   Otherwise → **plan-first**: present full plan, ask for confirmation, execute step-by-step.
4. **For each flow**: run user-gated separately before execution.

## Rules

- Do not execute any flow until all flows have been user-confirmed.
- Each completed flow must report before the next flow begins (sequential case).
- Parallel flows: report aggregated summary after all complete.
