# Performance & Resource Efficiency Rubric (SSOT)

> [!IMPORTANT]
> **Project Authority Priority Rule**: The host project's explicit architectural non-functional requirements (SLAs/SLOs, latency budgets, target database engines, and runtime constraints) **ALWAYS take precedence** over this rubric. This document serves as the foundational reference and quality baseline for semantic performance audits.

---

## 1. Persistence & Data Access Hygiene

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **$N+1$ Queries & Lazy Loading** | Accessing relational entity associations inside iteration loops (e.g., `records.map(r => r.profile.bio)`). | **Shopify**: Millions of recursive GraphQL queries saturated DB connection checkout pools. | Eager batch preloading (`includes`, `@EntityGraph`, DataLoader); enforce `strict_loading: true` in CI test suites. |
| **Cartesian Product Explosion** | Joining multiple independent collection relationships in a single SQL query (`Include(Items).Include(Shipments)`). | **Hibernate**: `MultipleBagFetchException`. Bypassing via `Set` triggers $N \times M \times K$ row multiplication, exhausting application heap. | **Split Queries**: Fetch primary collection via `JOIN`, batch secondary collections via separate `WHERE id IN (...)` queries (`AsSplitQuery()`, `@BatchSize(50)`). |
| **Unbounded Sets & Deep Offset** | Queries without non-nullable `LIMIT` or using high `OFFSET` (`LIMIT 50 OFFSET 100000`). | **Slack**: Storage engine scans and discards $100\text{k}$ disk pages, causing database timeouts and pagination drift. | **Keyset / Cursor Pagination**: Seek directly via indexed composite cursor `WHERE (created_at, id) < (?, ?) ORDER BY created_at DESC, id DESC LIMIT 50`. |
| **Transaction Wrapping Network I/O** | External HTTP calls, webhook dispatches, or slow RPCs enclosed within `@Transactional` blocks. | **Stripe**: Third-party payment latency spikes hold DB connections checked out, starving application connection pools (HikariCP/pgBouncer). | **Transactional Outbox / Atomic Phases**: Restrict DB transactions strictly to local ACID updates (< 2ms); dispatch external calls via asynchronous outbox workers. |

```text
Smell:  @Transactional void pay(Order o) { o.setPending(); repo.save(o); gateway.charge(o.total()); o.setPaid(); }
Remedy: Local transaction saves order + outbox task; background worker executes gateway.charge() without holding DB connection.
```

---

## 2. Memory Retention & Buffer Lifecycle

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Unbounded In-Memory Caches** | Using naked maps (`map[K]V`, `ConcurrentHashMap`) as caches without capacity limits or eviction. | **Discord**: Live-state pointer scanning caused Go GC latency spikes every 2 minutes even with low allocation rates. | Bounded cache using Window TinyLFU (**Caffeine** in Java, **Ristretto** in Go) with maximum size and TTL; offload to Redis if exceeding 500MB. |
| **Lifecycle & Scope Pinning** | `ThreadLocal` variables in worker pools without removal; closures capturing large enclosing scopes. | **Tomcat / V8**: Pooled threads pin application `ClassLoader`, leaking Metaspace permanently; V8 shared scopes leak massive byte strings. | Guaranteed cleanup via `try ... finally { threadLocal.remove(); }`; extract primitive IDs instead of capturing large object trees in closures. |
| **Allocation Churn & GC Pressure** | Allocating dynamic byte slices or large temporary structs repeatedly inside hot request loops. | **Cloudflare**: Allocating buffers per connection multiplied virtual memory 2.5x and forced high GC mark-assist overhead. | Buffer pooling via **`sync.Pool`** (Go) or **`ByteBuf`** (Netty); reset slice length before reuse (`b[:0]`) and guard against oversized buffers. |
| **In-Memory File/Blob Buffering** | Reading entire incoming payloads or files into memory (`io.ReadAll(r.Body)`, `file.getBytes()`). | **OOM Outage**: 50 concurrent 100MB file uploads consume 5GB heap memory, triggering Linux OOM-killer termination. | **Zero-Allocation Streaming**: Pipe streams directly to disk or cloud storage via bounded chunks (`io.Copy`, `InputStream.transferTo()`); use pre-signed direct upload URLs. |

```text
Smell:  byte[] data = io.ReadAll(request.Body); os.WriteFile(path, data);
Remedy: io.Copy(destinationFile, io.LimitReader(request.Body, MAX_UPLOAD_BYTES));
```

---

## 3. Concurrency Bottlenecks & Event-Loop Health

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Event-Loop Starvation** | Synchronous compute (`pbkdf2Sync`, heavy JSON parse, sync fs, ReDoS) on single-threaded event loops. | **Node.js / asyncio**: Event loop freezes for 200ms+; health-check probes fail, triggering cascading orchestrator pod restarts. | Offload CPU-heavy or blocking tasks to **`worker_threads`**, `asyncio.to_thread()`, or dedicated background services. |
| **Unbounded Thread Pool Queues** | Thread pools initialized with unbounded task queues (`Executors.newFixedThreadPool`, default queue). | **JVM Outage**: Latency spikes cause 100k+ tasks to queue invisibly; memory bloats until JVM terminates with `OutOfMemoryError`. | Size threads via Little's Law; use shallow bounded queues with explicit saturation policies: **`CallerRunsPolicy`** or fast 429/503 drop. |
| **Missing Stream Backpressure** | Writing unthrottled upstream data directly to slow downstream writers without checking drain events. | **Memory Blowout**: Fast producer overwhelms slow network socket, accumulating unconsumed chunks in RAM buffers until OOM. | Flow-controlled stream pipelines: **`stream.pipeline()`** (Node.js) or Reactive Streams protocol `request(n)`. |
| **Cache-Line False Sharing** | High-concurrency increments on a single shared atomic variable (`AtomicLong`, naked struct fields). | **Cloudflare Pingora**: Multi-core MESI bus contention thrashes L1/L2 cache lines, degrading throughput on 64+ core machines. | Cell-striped counters (**`LongAdder`** / `Striped64` in Java) or cache-line padding (`[64]byte` in Go/Rust). |
| **Unbounded Goroutine Spawning** | Spawning unmanaged `go func()` or task per request inside loops without concurrency gating. | **Scheduler Collapse**: Downstream latency creates 100k+ concurrent routines, exhausting heap and CPU scheduler bandwidth. | Gate concurrency using bounded buffered channel semaphores (`sem := make(chan struct{}, maxWorkers)`). |

```text
Smell:  for _, req := range requests { go handle(req) }
Remedy: Concurrency semaphore: acquire token before spawning, defer token release, monitor worker pool saturation.
```

---

## 4. Network & Distributed I/O Efficiency

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Chatty RPCs (Network N+1)** | Executing sequential network round-trips inside iterative loops (`for id in ids: client.get(id)`). | **Latency Cascade**: Total request latency equals $N \times \text{RTT}$; exhausts socket buffers and worker threads. | Standardize on **Batch RPCs** adhering to Google AIP-231 (`BatchGet[Resources](names: repeated string)`). |
| **Socket TIME_WAIT Exhaustion** | Instantiating short-lived HTTP clients per request (`new HttpClient()`) without connection pooling. | **Port Starvation**: Sockets linger in `TIME_WAIT` (60–120s), exhausting ephemeral ports (`EADDRNOTAVAIL`). | Reusable connection pools with HTTP Keep-Alive; set **`PooledConnectionLifetime`** (2–5 min) to honor DNS TTL updates. |
| **Cache Stampede / Thundering Herd** | Naive cache check falling back immediately to primary database query on every cache miss. | **Meta / Facebook**: Hot key expiration provokes thousands of concurrent database queries simultaneously, collapsing the DB. | **Singleflight Deduplication** (`singleflight.Group`) + **XFetch Probabilistic Early Expiration** ($-\beta \cdot \delta \cdot \ln(r)$). |
| **Heavy JSON Serialization Tax** | Loading multi-megabyte JSON payloads into RAM strings for reflection-based deserialization. | **Uber**: JSON serialization consumed >35% service CPU. Protobuf/gRPC reduced CPU consumption by 35% and P99 latency by 40%. | Adopt binary schemas (Protobuf/FlatBuffers) for microservice RPCs; use streaming decoders (`json.NewDecoder(r)`) for public REST. |

```text
Smell:  if (cache.get(key) == null) { data = db.query(key); cache.set(key, data); }
Remedy: In-flight singleflight coalescing collapses concurrent misses into 1 DB query; XFetch refreshes key before hard expiration.
```

---

## 5. Computational Hot-Paths & Algorithmic Debt

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Accidental $O(N \cdot M)$ Lookups** | Nested iterations executing linear scans (`list.contains()`, `array.find()`, `slice.Index()`). | **GTA Online**: Scanning growing JSON arrays $O(N^2)$ pegged CPU for 6+ minutes at startup (fixed to < 2 min). | Pre-index lookup targets into a **`HashSet` or `HashMap`** prior to iteration, reducing complexity to $O(N + M)$. |
| **Catastrophic Backtracking (ReDoS)** | Nested quantifiers or overlapping wildcards in regex (`(a+)+`, `.*.*=.*`) evaluated against user input. | **Cloudflare (2019 Outage)**: A single backtracking WAF regex pegged global edge CPUs at 100%, causing a 27-minute worldwide outage. | Deploy linear-time regular expression engines (**Google RE2** / Rust `regex`) guaranteeing $O(N)$ execution without backtracking. |
| **Invariant Compilation in Hot Loops** | Recompiling regular expressions or date formatters inside high-frequency loops (`regexp.MatchString`). | **Datadog**: Services consumed >85% of CPU compiling regex state machines rather than executing pattern matches. | **Loop-Invariant Hoisting**: Compile regular expressions and formatters once at module/class initialization (`MustCompile`, `static`). |
| **Quadratic String Concatenation** | Accumulating strings via `str += chunk` inside loops. | **GC Compaction**: Every concatenation copies the full existing buffer, generating $O(N^2)$ allocations and heavy GC thrashing. | Use pre-allocated mutable builders (**`StringBuilder(capacity)`**, **`strings.Builder.Grow()`**). |
| **Defensive Deep Copying** | Cloning large object trees (`DeepCopy()`, `.clone()`) on high-frequency read reconciles. | **Kubernetes**: Informer cache reconcile loops calling `DeepCopy()` on every read caused acute memory churn and GC spikes. | Pass read-only borrow references; enforce copy-on-write semantics strictly at mutation boundaries. |

```text
Smell:  String out = ""; for (String item : items) { out += item + "\n"; }
Remedy: StringBuilder sb = new StringBuilder(items.size() * 64); for (String item : items) { sb.append(item).append('\n'); }
```
