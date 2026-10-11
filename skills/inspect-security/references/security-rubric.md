# Software Security & Threat Exposure Rubric (SSOT)

> [!IMPORTANT]
> **Project Authority Priority Rule**: The host project's explicit architectural security policies, compliance mandates (PCI-DSS, HIPAA, GDPR, SOC2), threat models, and authorization frameworks **ALWAYS take precedence** over this rubric. This document serves as the foundational reference and quality baseline for semantic security audits.

---

## 1. Tenant Isolation & Object-Level Authorization (BOLA / IDOR)

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Ambient Identifier Trust (BOLA/IDOR)** | Fetching or mutating entities by client-supplied ID without verifying tenant ownership (`WHERE id = :id`). | **Uber**: API endpoints accepted arbitrary UUIDs (`restaurantUuid`, `employeeUuid`), allowing cross-account account takeover and profile leaks. | Compound lookup binding resource ID to authenticated session tenant: `WHERE id = :id AND tenant_id = :tenant_id`. |
| **Direct URL / Nested Resolver Traversal** | Exposing child resources in REST or GraphQL child resolvers without cascading tenant context. | **Shopify**: GraphQL invoice traversal leaked merchant billing data because parent-level auth did not cascade to nested node resolvers. | Enforce authorization predicates at every graph node / child resolver; propagate immutable `TenantContext`. |
| **Shared Connection State Bleed** | Setting tenant context on pooled database connections without transaction scoping (`SET app.tenant_id = ?`). | **PgBouncer Outage**: Contaminated pooled sessions executed queries from Tenant B within Tenant A's session context. | Enforce database-level **PostgreSQL Row-Level Security (RLS)** using transaction-scoped settings: `SET LOCAL app.current_tenant_id = :id`. |
| **Client-Controlled Tenancy Headers** | Deriving tenant context from unverified client headers (`X-Tenant-ID`) or URL query parameters. | Multi-tenant SaaS tenant hopping; attackers forge header to access sibling tenant data. | Derive tenant and user identity strictly from verified server-side session or cryptographically verified JWT claims. |

```text
Smell:  db.query("SELECT * FROM invoices WHERE id = $1", [req.params.id]);
Remedy: db.query("SELECT * FROM invoices WHERE id = $1 AND tenant_id = $2", [req.params.id, session.tenantId]);
```

---

## 2. Authentication Integrity & Token Lifecycle

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Dynamic Algorithm Acceptance** | Verifying JWT signatures using header-declared `alg` without an immutable allowlist. | **CVE-2015-9235 (Auth0 / PyJWT)**: Attackers switched `RS256` to `HS256`, signing forged tokens with public keys as HMAC secrets. | Explicit algorithm whitelisting: `algorithms: ['RS256']`. Reject `none`, symmetric keys on asymmetric routes, and untrusted key headers (`jwk`, `jku`). |
| **Unbounded Token Lifespan & Skew** | Bearer access tokens with days/weeks lifetime or permissive clock skew (> 60s). | **Okta / Cloudflare (2023)**: Harvested bearer tokens from support HAR logs replayed across enterprise tenants. | Short-lived access tokens (5–15 min max), clock skew $\le 60$s, edge revocation cache (`jti` / `revoked_before` TTL matching token duration). |
| **Replay-Vulnerable Refresh Tokens** | Refresh tokens updated without replay detection or single-use consumption. | Session hijacking; stolen refresh tokens grant indefinite access even after user logs out. | **Token Family Tracking (RFC 9700 §4.14.2)**: Single-use rotation; reusing an invalidated token cascades instant revocation across the entire family tree. |
| **Permissive OAuth Redirects & Missing PKCE** | Using regex/wildcards for OAuth `redirect_uri` or omitting S256 PKCE in client flows. | **PortSwigger OAuth Labs**: Directory traversal (`/callback/../../redirect`) leaks authorization codes to attacker domains. | Exact character-by-character string matching for redirect URIs (RFC 9700 §4.1.3); universal `code_challenge_method=S256` (RFC 7636). |
| **ECDSA Zero Boundary Omission** | Incomplete range check on signature components $r, s$ before elliptic curve mathematical evaluation. | **CVE-2022-21449 ("Psychic Signatures")**: Blank signatures ($r=0, s=0$) in Java 15–18 evaluated to $0 = 0$, validating any token for any key. | Assert mathematical invariant $0 < r < n$ and $0 < s < n$ prior to cryptographic verification. |

```text
Smell:  jwt.verify(token, publicKey); // Allows attacker-supplied alg: "HS256" or "none"
Remedy: jwt.verify(token, publicKey, { algorithms: ['RS256'], issuer: 'auth.domain', audience: 'api.domain', clockTolerance: 30 });
```

---

## 3. Semantic Injection & Taint Sinks

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Raw ORM Escape Hatches** | String interpolation inside ORM raw query interfaces (`$queryRawUnsafe`, `whereRaw`, raw HQL). | **TypeORM (CVE-2025-60542)**: Raw clause construction bypassed query-builder parameterized boundaries, allowing direct SQL injection. | Tagged template literals (`prisma.$queryRaw\`SELECT * FROM t WHERE id = ${id}\``) or strict parameter arrays (`$1, $2`). |
| **NoSQL Operator Smuggling** | Passing unvalidated request bodies directly to MongoDB/Mongoose filters (`db.find(req.body)`). | **CWE-943**: Attackers submit `{"password": {"$ne": null}}` or `$regex` payloads, bypassing authentication and exfiltrating data. | Strict schema validation (Zod `.strict()`, Pydantic `extra='forbid'`) enforcing primitive strings before query dispatch. |
| **Server-Side Template Injection (SSTI)** | Interpolating user input into template source strings rather than passing context variables. | **Uber (HackerOne #125980)**: Dynamic Jinja2 email templates rendered user profile names, achieving unauthenticated RCE. | Immutable static filesystem templates; user variables passed strictly as data dictionaries within sandboxed engines. |
| **JavaScript Prototype Pollution** | Recursive deep object merging without property allowlists or prototype protections. | **Kibana (CVE-2019-7609) / Lodash**: Insecure merge mutated `Object.prototype`, creating gadgets leading to command execution. | Filter keys (`__proto__`, `constructor`, `prototype`); use `Object.create(null)` or `Map`; apply `Object.freeze(Object.prototype)`. |

```text
Smell:  await prisma.$queryRawUnsafe(`SELECT * FROM users WHERE email = '${req.body.email}'`);
Remedy: await prisma.$queryRaw`SELECT * FROM users WHERE email = ${req.body.email}`;
```

---

## 4. Server-Side Request Forgery (SSRF) & Network Pivoting

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Cloud Metadata Exposure (IMDSv1)** | Unauthenticated HTTP GET access to cloud metadata IP (`169.254.169.254`). | **Capital One ($190M Breach)**: WAF SSRF queried IMDSv1, extracting temporary IAM credentials with wildcard access across 700+ S3 buckets. | Enforce **AWS IMDSv2** globally (`HttpTokens: required`, `HttpPutResponseHopLimit: 1`); block metadata IP via host `iptables` rules. |
| **TOCTOU DNS Rebinding** | Resolving domain to check IP in step 1, then fetching original URL in step 2. | **GitLab SSRF**: DNS server with TTL=0 returned public IP during validation and `127.0.0.1` during request, hitting internal Redis. | **Single-Resolution Validate-and-Pin**: Resolve DNS once, validate IP against private subnets, connect socket directly to validated IP. |
| **Alternative IP Encoding Bypasses** | Blacklisting IPs with naive string matching or regex (`"127.0.0.1"`, `"localhost"`). | Bypassed by dword (`2130706433`), hexadecimal (`0x7f000001`), octal (`0177.0.0.1`), IPv6-mapped (`[::ffff:127.0.0.1]`), or `0.0.0.0`. | Parse strings with canonical binary/numeric IP parsers (Go `netip`, Python `ipaddress`); evaluate against CIDR block sets. |
| **Direct Application Egress** | Production web servers possessing direct outbound Internet connectivity to fetch webhooks/avatars. | Lateral movement from compromised pods into internal VPC microservices, databases, and message queues. | Dedicated, mTLS-authenticated Egress Proxy (**Stripe Smokescreen** or **Envoy**) in isolated DMZ with zero routes to VPC subnets. |

```text
Smell:  if is_public(dns.resolve(url)): requests.get(url) // TOCTOU DNS Rebinding vulnerability!
Remedy: Resolve once -> parse canonical IP -> assert not in RFC1918/Loopback/Metadata -> dial socket directly to IP with Host header.
```

---

## 5. Concurrency Races & Financial Consistency

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Application Check-Then-Act (TOCTOU)** | Querying balance in application memory, verifying balance, then updating (`if bal >= amt: update`). | **Flexcoin ($600K Bankruptcy)**: Parallel worker threads read identical starting balances before writes committed, overdrawing 896 BTC. | **Atomic Conditional SQL**: `UPDATE accounts SET balance = balance - :amt WHERE id = :id AND balance >= :amt` (verify `rows_affected == 1`). |
| **Multi-Resource Lock Inversion** | Locking transfer parties in arbitrary order or without pessimistic row locks. | **Starbucks Gift Card Race / Deadlocks**: Concurrent transfers between User A and B caused deadlocks or balance multiplication. | Deterministic row sorting with pessimistic locks: `SELECT id FROM accounts WHERE id IN (:a, :b) ORDER BY id ASC FOR UPDATE`. |
| **Single-Packet HTTP/2 Limit Overrun** | Enforcing single-use promos or checkout quotas via uncoordinated application flags. | **PortSwigger Research**: 20–30 parallel requests synchronized into a single packet execute within sub-millisecond windows. | Atomic reservation flagging within the database transaction; require client `Idempotency-Key` with unique database constraint. |
| **Unfenced Distributed Locks** | Relying on Redis lock TTLs (Redlock) for mutual exclusion during long transactions. | **Kleppmann Proof**: GC pauses or network hiccups expire Redis TTL; two workers commit overlapping writes. | Monotonic **fencing tokens** verified at the storage engine: `WHERE last_fencing_token < :incoming_token`. |

```text
Smell:  acc = db.get(id); if (acc.balance >= amt) { db.set(id, acc.balance - amt); }
Remedy: res = db.execute("UPDATE accounts SET balance = balance - :a WHERE id = :id AND balance >= :a", a=amt, id=id); if (res.rowcount == 0) throw OverdraftError();
```

---

## 6. Cryptographic Misconfigurations & Secret Hygiene

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Insecure PRNG (CWE-338)** | Using non-cryptographic random generators (`Math.random()`, `rand.Int()`) for tokens or nonces. | **NodeBB (CVE-2022-35255)**: PRNG internal state reconstructed after observing consecutive tokens, enabling account takeover. | Derive entropy exclusively from operating system CSPRNG (`crypto.randomBytes`, `crypto/rand`, Python `secrets`). |
| **Fast / Non-Memory-Hard Hashing** | Storing passwords using MD5, SHA-1, SHA-256, or un-salted single-round SHA-512. | **Adobe (153M Passwords)**: Fast hashes cracked at billions of guesses/sec using commodity GPU rigs. | **Argon2id (RFC 9106)** ($m=19\,\text{MiB}, t=2, p=1$); fallback to bcrypt (work factor $\ge 12$) or PBKDF2 ($\ge 600{,}000$ iterations). |
| **Timing Side Channels on Secrets** | Using standard string comparison (`===`, `==`, `strcmp`) to verify API keys, HMACs, or reset tokens. | Single-packet HTTP/2 timing attacks reveal matching secret characters byte-by-byte in sub-millisecond windows. | **Constant-Time Pre-Hashed Comparison**: Pre-hash both inputs to 32-byte SHA-256, then call `crypto.timingSafeEqual` (prevents length oracles). |
| **Pattern-Preserving Ciphers** | Encrypting data in Electronic Codebook (ECB) mode or unauthenticated CBC mode. | **Adobe (2013)**: Identical plaintext blocks yielded identical ciphertext blocks, preserving data structures and patterns. | Authenticated Encryption with Associated Data (AEAD): **AES-256-GCM** (96-bit unique IV) or **ChaCha20-Poly1305** (RFC 8439). |
| **Hardcoded Secrets (CWE-798)** | Hardcoding API credentials, private keys, or webhook signing secrets in source code. | **Uber / Toyota Disclosures**: Embedded AWS/DB credentials committed to VCS exfiltrated to breach customer data. | Inject secrets at runtime via cloud secret managers (Vault/KMS); enforce pre-commit shift-left scanners (`gitleaks protect`). |

```text
Smell:  if (userToken === actualSecret) { ... } // Timing attack & length oracle
Remedy: timingSafeEqual(crypto.createHash('sha256').update(userToken).digest(), crypto.createHash('sha256').update(actualSecret).digest());
```

---

## 7. Mass Assignment & Workflow State Integrity

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Mass Assignment / Over-Posting (CWE-915)** | Binding raw HTTP payloads (`req.body`, `params`) directly to database models or domain entities. | **GitHub Breach (Egor Homakov)**: Mass assignment on public key upload form injected attacker key into `rails/rails` org. | Strict input Transfer Objects (DTOs) with field allowlists (NestJS `ValidationPipe(whitelist=true)`, Zod `.strict()`, Pydantic). |
| **State Machine Skipping** | Permitting arbitrary status mutations via CRUD endpoints (`PATCH /orders { status: "PAID" }`). | E-commerce state skipping: Attackers jump from `ORDER_CREATED` directly to `ORDER_PAID` without invoice settlement. | Formal Finite State Machine (FSM) transition guards; enforce cryptographic event proofs before mutating lifecycle states. |
| **Parameter Tampering & Price Manipulation** | Trusting client-supplied prices, discount percentages, or negative item quantities. | Pricing bypasses: Submitting `quantity: -1` offsets cart totals, purchasing products for $0.00. | Server derives prices authoritatively from database; enforce schema constraints (`quantity >= 1`, integer cents). |

```text
Smell:  user.update(req.body); // Attacker injects: { "is_admin": true, "credit_balance": 999999 }
Remedy: const dto = UpdateUserDto.parse(req.body); user.update({ displayName: dto.displayName, bio: dto.bio });
```

---

## 8. Data Privacy, PII Scrubbing & Error Decoupling

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Sensitive Data in Application Logs (CWE-532)** | Logging raw request bodies, auth headers, credit card PANs, or passwords to stdout / APM. | **Twitter / Meta Disclosures**: Raw user passwords printed to internal diagnostic logs prior to hashing. | In-process structured logger path redaction (`pino.redact: ['req.headers.authorization', 'req.body.password']`); log pipeline scrubbers. |
| **Verbose Diagnostic Error Disclosure (CWE-209)** | Returning raw database exceptions, stack traces, or internal server errors directly to clients. | Attackers glean database table structures, SQL syntax dialects, internal IP topology, and dependency versions. | **RFC 9457 Problem Details**: Client receives sanitized problem JSON with opaque `correlation_id`; full stack traces kept internal. |
| **PII Leakage in Default Serialization** | Serializing database entities directly to JSON HTTP responses without output projection. | Responses leak `password_hash`, `mfa_secret`, `ssn`, and internal tenant notes to API consumers. | Fail-closed response DTOs or entity serialization decorators (`@Exclude()`, explicit mapper functions). |
| **Secrets in URL Query Strings** | Accepting authentication tokens or reset secrets via GET query parameters (`?token=...`). | **GitHub Brownout**: Secrets in query strings recorded verbatim in proxy access logs, browser history, and `Referer` headers. | Transmit credentials exclusively via `Authorization: Bearer` headers or secure, SameSite request bodies. |

```text
Smell:  res.status(500).json({ error: err.message, stack: err.stack });
Remedy: logger.error({ err, traceId }); res.status(500).type('application/problem+json').json({ title: "Internal Error", instance: `urn:trace:${traceId}` });
```

---

## 9. Client Boundary Hardening & CORS/CSP

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Permissive CORS Origin Reflection (CWE-942)** | Dynamically reflecting `Origin` header with `Access-Control-Allow-Credentials: true` or loose regex. | **Uber / Steam Disclosures**: Regex matching `uber.com.attacker.com` reflected origins, allowing authenticated credential theft. | Explicit origin allowlists evaluated against immutable Sets; emit `Vary: Origin`; never reflect arbitrary origins with credentials. |
| **Trusting Origin: null** | Whitelisting `Origin: null` in CORS configuration to accommodate local files or testing. | Attackers execute cross-origin requests from sandboxed iframes (`<iframe sandbox="allow-scripts">`), which send `Origin: null`. | Explicitly reject `Origin: null` from credentialed CORS endpoints. |
| **Absence of Strict Content Security Policy** | Missing CSP or using `'unsafe-inline'`, `'unsafe-eval'`, and broad wildcard domain allowlists. | **British Airways Magecart**: Missing strict script integrity and CSP allowed malicious scripts to exfiltrate payment card data. | Nonce-based Strict CSP Level 3: `script-src 'nonce-<rand>' 'strict-dynamic'; object-src 'none'; base-uri 'none'; frame-ancestors 'none';`. |
| **State Mutation via Idempotent Methods / Missing CSRF** | Executing state-changing mutations (delete, transfer) via GET or without anti-CSRF synchronizer tokens. | Cross-Site Request Forgery: Top-level link navigation bypasses `SameSite=Lax` cookie protections, executing unwanted mutations. | Enforce POST/PUT/DELETE for all mutations; validate anti-CSRF synchronizer tokens or custom headers (`X-CSRF-Token`); set cookies `SameSite=Strict`. |
| **Missing HTTP Transport Security Headers** | Serving HTTP responses without HSTS, nosniff, or framing protection headers. | SSL stripping attacks, MIME-type sniffing attacks, and UI Clickjacking in malicious framing contexts. | Enforce baseline headers: `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`, `X-Content-Type-Options: nosniff`. |

```text
Smell:  if (origin.includes("example.com")) { res.setHeader('Access-Control-Allow-Origin', origin); res.setHeader('Access-Control-Allow-Credentials', 'true'); }
Remedy: if (ALLOWED_SET.has(origin)) { res.setHeader('Access-Control-Allow-Origin', origin); res.setHeader('Access-Control-Allow-Credentials', 'true'); } res.setHeader('Vary', 'Origin');
```

---

## 10. Application-Layer Resource Exhaustion & Rate Limiting

| Risk Category | Observable Code Smell | Production Failure Mode (Battle Scar) | Standard Production Remedy |
|---|---|---|---|
| **Missing Semantic Rate Limiting (CWE-770)** | Exposing authentication, search, or OTP dispatch endpoints without multi-key rate limiters. | **Telco IRSF Toll Fraud / Brute-Force**: Botnets abusing SMS OTP endpoints incurred thousands of dollars in toll fraud within minutes. | Multi-dimensional rate limiters (IP, authenticated User/Org, device, destination prefix); SMS daily spending caps and conversion tripwires. |
| **Fixed-Window Spike Traps** | Using naive fixed-window counters for rate limiting (`INCR` with basic expiry). | Traffic bursts at window boundaries allow 2x configured traffic volume, overwhelming backend services. | **Sliding Window Counter (Cloudflare Model)** via Redis atomic Lua scripts or token buckets. |
| **GraphQL Depth & Complexity Exhaustion** | Unbounded nested cyclical queries (`author -> posts -> author -> posts...`) on GraphQL schemas. | **Shopify / GitHub**: Cyclical queries spawned thousands of nested database sub-queries, exhausting connection pools and memory. | Pre-execution AST validation rules enforcing maximum query depth (6–10) and static query complexity point budgets. |
| **Decompression Bombs (Zip / XML Bombs)** | Reading archives or untrusted XML payloads without stream allocation limits. | **Billion Laughs / 42.zip**: Small kilobyte payloads expand into gigabytes in RAM, triggering instant Linux OOM-killer termination. | Wrap decompression streams in bounded readers (`io.LimitReader`, `http.MaxBytesReader`); enforce `defusedxml` with `forbid_dtd=True`. |
| **Unbounded Payload Body Caps** | Missing `Content-Length` ceilings or streaming buffers on API gateways and handlers. | Slow-drip or massive chunked requests exhaust web server connection pools and heap memory. | Configure reverse proxy caps (`client_max_body_size 2M`) and handler streaming limits (`http.MaxBytesReader`). |

```text
Smell:  data, err := io.ReadAll(zipFileStream) // OOM if payload is decompression bomb!
Remedy: data, err := io.ReadAll(io.LimitReader(zipFileStream, MAX_BYTES + 1)); if len(data) > MAX_BYTES throw PayloadTooLarge();
```
