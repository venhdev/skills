# Examples

## Example 1 — Simple (1 Overview)

Req: "Diagram `login_screen.dart` — email/password form."

Parse: source present; target/zoom/mode missing → ASK.
SPEC ×2: target=Inline; zoom=Overview.
Generate: `login-flow | flowchart | end-to-end?`
Confirm: 1 diagram. Approve.
Pre-render: mmdc missing → Inline.

→ 2 user turns. 1 flowchart.

## Example 2 — Complex (3 Detail)

Req: "Diagram `checkout_service.dart` — cart, payment, shipping, retry. Target: `<req_dir>/diagrams/`."

Parse: target present; zoom/mode missing → ASK.
SPEC ×2: zoom=Detail; mode=Inline+SVG.
Generate:
- `checkout-flow | flowchart | end-to-end?`
- `payment-fsm | stateDiagram-v2 | what states?`
- `checkout-api | sequenceDiagram | who calls whom?`
Confirm: 3 diagrams + planned SVG paths. Approve.
Pre-render: mmdc available, SVG, target empty.

→ 2 user turns. 3 SVGs.

## Example 3 — Trivial (fast path)

Req: "Diagram `utils.dart` — 1 helper `formatDate()`."

Parse: trivial → auto-default Overview + Inline. SPEC: SKIP.
Generate: 1 flowchart.
Confirm + Pre-render: Inline.

→ 1 user turn. 1 flowchart.

## Anti-patterns (blockers)

❌ Asking inputs already in req
❌ Generating before SPEC gate
❌ Inventing types outside the 6 allowed
❌ Combining static structure + runtime behavior in one diagram
❌ Auto-installing mmdc or Chromium
