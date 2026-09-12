# The Invariant Dimension Rubric

Authoritative testing vectors for evaluating system reliability beyond superficial line coverage.

## Dimension Matrix

| Dim | Name | What Must Be Proven | Common Blind Spot |
| :--- | :--- | :--- | :--- |
| **D1** | **Baseline (Happy Path)** | Core workflow produces valid output and state transition with valid inputs. | Assuming happy path proves full system correctness. |
| **D2** | **Boundary & Extremes** | System handles zero, empty collections, maximum integer bounds, and extreme string lengths. | Testing only nominal inputs (e.g., standard collections, average strings). |
| **D3** | **Negative & Invalidation** | Malformed inputs and forbidden operations return typed errors without panicking. | Catching generic exceptions or ignoring error body contracts. |
| **D4** | **Concurrency & Idempotency** | Simultaneous racing operations or duplicate retry requests produce deterministic, race-free state. | Assuming operations run sequentially in single-threaded isolation. |
| **D5** | **Lifecycle & State Transitions** | State transitions strictly follow legal state machine graphs (e.g., Draft -> Active -> Archived); invalid state jumps are rejected. | Mutating state fields directly in database without going through lifecycle guards. |
| **D6** | **Time Travel & Expiry** | Timestamps, TTL expirations, and timezone shifts function identically across UTC boundaries. | Testing only with current time; ignoring tokens or sessions expiring midway. |
| **D7** | **Nullability & Schema Drift** | System tolerates omitted optional fields, legacy payload versions, and partial sync updates. | Assuming client and server always run identical, up-to-date schema versions. |
| **D8** | **Perimeter & Tenant Isolation** | Data access boundaries strictly prevent cross-tenant leakage or unauthorized privilege escalation. | Testing logic with a single superuser, ignoring tenant scoping. |
| **D9** | **Degradation & Failure Recovery** | Transient I/O drops, storage timeouts, and retry backoffs recover gracefully without corrupting state. | Assuming external dependencies and network connections are reliable. |

## Dimension Audit Queries

When auditing a test suite for a component, interrogate:
1. *Is there an idempotency test if this endpoint/mutation can be retried?* (D4)
2. *Can this entity be forced into an illegal state transition?* (D5)
3. *What happens to this logic across year or day boundaries in UTC?* (D6)
4. *Can an entity be accessed across tenant boundaries by guessing its identifier?* (D8)
