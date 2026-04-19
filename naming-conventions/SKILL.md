---
name: naming-conventions
description: Use this skill whenever naming a new service, feature, module, Docker container, env var prefix, class, type, interface, enum, file, function, test, JSON key, DB table/column, gRPC proto, or any new identifier. Trigger when the user asks to add a service, create a feature, rename something, scaffold a file, define a type, write a test, or create a DB table. Also trigger proactively when a new name could collide with existing ones. This skill prevents ambiguous, overloaded, or YAGNI-violating names using a per-layer decision tree + universal anti-pattern checklist. Make sure to use this skill on every new name — the cost of a bad name is permanent.
---

# Naming Conventions

Naming is the most permanent decision in a codebase. A bad name survives refactors, team changes, and cloud migrations.

## Universal Anti-Pattern Checklist

Apply this to **every name** before settling on it:

```
❌  NEVER encode:  tech stack    (node, nestjs, apollo, fcm)
❌  NEVER encode:  team name     (payments-team, platform-team)
❌  NEVER encode:  region        (us-east, eu-west, ap-south)
❌  NEVER encode:  environment   (prod, staging, dev, local)
❌  NEVER encode:  version       (v2, v3) — use URL path / header
❌  NEVER use:     I-prefix on interfaces (IUserService) — TS structural types don't need it
❌  NEVER use:     camelCase filenames for classes/types
❌  NEVER use:     snake_case for class/interface names
❌  NEVER mix:     plural/singular inconsistently in same category
```

---

## Layer 1: Service & Architecture Names

**Name the capability, not the implementation.**

**Decision tree:**

```
BOUNDED-CONTEXT SERVICE?
  └── No → MODULE within existing service (feature folder naming → Layer 3)
  └── Yes → MULTIPLE CONSUMERS?
       └── Yes → encode consumer: storefront-api, partner-api, internal-api
       └── No → PROTOCOL NEED EXPLICIT?
            └── Yes → encode dominant protocol: api-gateway, graphql-bff
            └── No → IS IT A BFF? → suffix -bff : suffix -service
```

| Category | Pattern | Examples |
|---|---|---|
| Domain services | `<domain>-service` | auth-service, payment-service, notification-service |
| Gateways / BFFs | `<protocol>[-<consumer>]-<layer>` | api-gateway, graphql-bff, mobile-bff |
| Client apps | `<consumer>-portal` | web-portal, admin-panel |
| Infrastructure | `<type>` (no domain) | postgres, redis, traefik |
| Workers / Jobs | `<domain>-worker` | location-cleanup-worker |

**Docker field mapping:**

| Field | Convention | Example |
|---|---|---|
| Compose service | kebab-case DNS hostname | `api-gateway` |
| Docker image | `<org>-<service-name>` | `myapp-api-gateway` |
| Container name | `<org>-<service-name>` | `myapp-api-gateway` |
| Env file | `env/<service-name>.env` | `env/api-gateway.env` |

---

## Layer 2: Class, Interface, Type & Enum Names (PascalCase)

### Class / Interface — no I-prefix

TypeScript interfaces and classes are structurally equivalent. I-prefix breaks symmetry and is discouraged by the TS Handbook, Airbnb, Angular, and NestJS.

```typescript
// ✅ Correct
interface UserService { findById(id: string): Promise<User | null>; }
class DefaultUserService implements UserService { ... }

// ❌ Never — redundant I-prefix
interface IUserService { ... }
```

### Type Aliases & Parameters

```typescript
type UserRole = 'admin' | 'user' | 'guest';           // union → type, not interface
interface Mapper<TInput, TOutput> { map(i: TInput): TOutput; }  // generic params
```

### Enums — two styles, pick one per project

| Style | Enum | Members | Use when |
|---|---|---|---|
| TS Handbook | `PascalCase` | `PascalCase` | Pure TS, no DB mapping |
| Prisma/TypeORM | `PascalCase` | `SCREAMING_SNAKE_CASE` | DB-mapped |

```typescript
enum OrderStatus { Pending, Processing, Shipped, Delivered }    // Style A
enum OrderStatus { PENDING, PROCESSING, SHIPPED, DELIVERED }  // Style B
const Status = { Active: 'ACTIVE' } as const;                 // ✅ const enum alternative
```

### DTO Suffix — `Dto` (PascalCase)

```
✅  CreateUserDto   UpdateUserDto   UserResponseDto   LoginRequestDto
❌  userDTO        create_user_dto
```

### Domain Events — past tense + Event suffix

```
✅  UserCreatedEvent   PaymentReceivedEvent   OrderDeliveredEvent
❌  CreateUserEvent    (present tense — event already happened)
```

### Error Classes — `Error` / `Exception` suffix

```
✅  InvalidTokenError   PaymentFailedError   ConflictException
❌  TokenError         (too vague — what kind?)
```

---

## Layer 3: File & Directory Names

**Core rule:** `PascalCase.ts` for single class/type exports. `kebab-case.ts` for utilities, DTOs, enums, and multi-symbol files.

### File naming by content type

| Content | Convention | Example |
|---|---|---|
| Single class/type/interface | `PascalCase.ts` | `UserService.ts` |
| DTO / Entity / Enum | `kebab-case.<kind>.ts` | `create-user.dto.ts`, `user-role.enum.ts` |
| Utility / function files | `kebab-case.ts` | `date-utils.ts` |
| Type-only files | `kebab-case.types.ts` | `api.types.ts` |
| Unit tests | `PascalCase.spec.ts` | `UserService.spec.ts` |
| E2E tests | `kebab-case.e2e.ts` | `auth.e2e.ts` |
| NestJS modules | `PascalCase.module.ts` | `UsersModule.ts` |

### Directory naming — `kebab-case` everywhere

```
✅  src/features/auth/   clients/web-portal/   src/shared/ui/
❌  src/modules/sessionCleanup/   (camelCase)
❌  src/Modules/SessionCleanup/   (PascalCase)
```

### NestJS module directories — **singular** noun

```
✅  src/users/   ❌  src/user/   ❌  src/userS/
```

URL path is plural (`/users`) but module folder is singular — the module represents the concept, not the collection.

### Feature folder structure

```
src/
├── features/
│   ├── auth/
│   │   ├── auth.controller.ts     ← PascalCase: one class per file
│   │   ├── auth.service.ts
│   │   ├── auth.module.ts
│   │   ├── dto/                   ← kebab-case directory
│   │   │   └── create-auth-token.dto.ts
│   │   └── guards/
│   │       └── jwt-auth.guard.ts
│   └── payments/
└── shared/
    └── utils/                     ← no barrel files here; import directly
```

**Barrel files (`index.ts`)** — use only at package/module boundaries:
```
✅  packages/contracts/src/index.ts     (package public API)
❌  shared/utils/index.ts               (forces implicit coupling)
```

### Next.js reserved filenames — always `lowercase.tsx`

```
✅  app/layout.tsx   app/page.tsx   app/users/[id]/route.ts
❌  app/Layout.tsx  app/Page.tsx
```

---

## Layer 4: Function & Method Names (camelCase)

### Verb prefixes

| Prefix | Semantics | Example |
|---|---|---|
| `get` | Idempotent retrieve | `getUserById(id)` |
| `find` | Search with predicate | `findUsersByRole(role)` |
| `fetch` | Async retrieval (network/DB) | `fetchOrdersFromAPI()` |
| `create` | Produce new resource | `createUserSession()` |
| `set` / `patch` | Explicit mutation | `setCurrentLocale(loc)`, `patchUserEmail()` |
| `update` | Modify existing resource | `updateUserProfile(id, patch)` |
| `delete` / `remove` | Destroy resource | `removeRefreshToken(tid)` |
| `compute` / `calculate` | Derive value from inputs | `calculateProratedRefund()` |
| `validate` | Return boolean | `validatePasswordStrength()` |
| `sanitize` | Strip dangerous content | `sanitizeHtml(raw)` |
| `build` | Assemble composite objects | `buildGrpcMetadata(ctx)` |
| `parse` / `serialize` | String ↔ structured type | `parseAuthorizationHeader()` |

### Predicate functions — `is` / `has` / `can` / `should`

Return `boolean`, never throw.

```
✅  isAuthenticated(ctx)   hasPermission(u, p)   canAccessResource(u, r)
❌  checkUserExists(e)   validateEmail(e)       (use is/has instead)
```

### Event handlers — `on` / `handle` / `emit`

```
✅  onUserCreated(p)   handleIncomingMessage(m)   emitUserSignedUp(u)
```

---

## Layer 5: JSON Keys & API Payloads

**Standard: `camelCase` + plural nouns for arrays.**

```json
{
  "userId": "uuid",         "displayName": "Alice",
  "roles": ["admin"],       "createdAt": "2026-03-29T00:00:00Z",
  "lineItems": [...],       "shippingAddress": { "street": "..." }
}
```
```
❌  "user_id"   "userIdList"   "orderItem": [...]   (singular array name)
```

---

## Layer 6: Environment Variables

**Pattern: `<SERVICE>_<CONCEPT>` — UPPER_SNAKE_CASE**

Every service-specific env var **must** have a `<SERVICE>_` prefix to prevent collision in monorepos.

```
✅  NOTIFICATION_RABBITMQ_URL   AUTH_JWT_SECRET   USER_DB_HOST
✅  NEXT_PUBLIC_GATEWAY_URL    NODE_ENV
❌  RABBITMQ_URL   SMTP_HOST   GRPC_PORT   FCM_PROJECT_ID   API_KEY
```

---

## Layer 7: Database Schema Names

### PostgreSQL — `snake_case`, plural tables, singular columns

```sql
-- Tables: plural
✅  users   order_items   user_sessions

-- Columns: singular
✅  id   user_id   created_at   email_encrypted   deleted_at

-- Prisma: PascalCase model, @map() to snake_case DB
model User {
  id        String   @id @default(uuid()) @map("id")
  email     String   @unique                @map("email_encrypted")
  firstName String                        @map("first_name")
  createdAt DateTime @default(now())     @map("created_at")
  @@map("users")
}
```

---

## Layer 8: gRPC / Protobuf Names

```
✅  package myapp.notification.v1;           ← lowercase, versioned
❌  package MyApp.Notification.V1;        ← PascalCase

✅  rpc GetUser(GetUserRequest) returns (User);   ← PascalCase VerbNoun
❌  rpc get_user(GetUserRequest) returns (User);

✅  message User {                          ← PascalCase message
      string display_name = 3;            ← proto fields: camelCase
      repeated string roles = 4;
    }

✅  enum OrderStatus {
      ORDER_STATUS_UNSPECIFIED = 0;         ← REQUIRED: zero value
      ORDER_STATUS_PENDING = 1;
    }
```

---

## Layer 9: Feature Flags

**Pattern: `{category}.{feature-name}` — kebab-case**

| Prefix | Purpose | Example |
|---|---|---|
| `feature-` | General rollout | `feature-new-checkout-flow` |
| `experiment-` | A/B test | `experiment-homepage-v2-a-b` |
| `release-` | Dark launch | `release-graphql-bff-v2` |
| `killswitch-` | Emergency disable | `killswitch-legacy-payment` |
| `ops-` | Operational | `ops-maintenance-mode` |

---

## Layer 10: Git Branch & Commit Names

**Branches: `type/TICKET-description`**

```
✅  feature/MEMO-123-add-notification-service
✅  fix/MEMO-456-login-redirect-loop
✅  chore/upgrade-nestjs-v11
❌  new-feature   MEMO-123   bugfix
```

**Commits: Conventional Commits `type(scope): imperative`**

```
✅  feat(auth): add OAuth2 PKCE login flow
✅  fix(checkout): resolve null ref on empty cart
✅  refactor(location): extract compute logic to domain service
✅  feat(payments)!: introduce Stripe v2 webhook (BREAKING)
❌  Updated stuff   Fix bug   WIP
```

---

## Output Format

Produce a **Naming Decision Report** for every naming decision:

```markdown
## Naming Decision Report

**Proposed name:** `<name>`
**Layer:** [service | class | file | function | env-var | db-table | proto | json-key | flag | branch]

**Decision path:** [which branch of the decision tree]
**Anti-pattern check:** [x] No tech stack · [x] No team · [x] No region · [x] No env · [x] No version · [x] Correct case

**Alternatives considered:**
| Name | Reason rejected |

**Collision:** [searched codebase — no collision / found: X]
**Files to update:** [if renaming]
```

---

## Quick Reference Card

```
SERVICE:     <domain>-service | <protocol>[-<consumer>]-bff    (kebab-case)
CLASS:       PascalCase, no I-prefix, Service/Controller/Dto/Entity/Event/Error suffix
TYPE:        PascalCase, no I-prefix, T<T> or PascalCase for params
ENUM:        PascalCase, members: PascalCase or SCREAMING_SNAKE_CASE
FILE:        PascalCase.ts (single class) | kebab-case.<kind>.ts (DTO/enum/utils)
FUNCTION:    camelCase, verb prefix (get/find/create/validate/is/has/can)
ENV VARS:     UPPER_SNAKE_CASE, <SERVICE>_<CONCEPT>
JSON KEYS:   camelCase, plural arrays
DB TABLES:   snake_case, plural    (users, order_items)
DB COLUMNS:  snake_case, singular (user_id, created_at)
PRISMA:      PascalCase model, @map("snake_case")
PROTO PKG:   lowercase.v1         (myapp.notification.v1)
PROTO RPC:   PascalCase VerbNoun  (GetUser, CreateOrder)
ENUM VAL:    SCREAMING_SNAKE_CASE (ORDER_STATUS_PENDING)
FLAG:        kebab-case, category prefix (feature-new-checkout)
BRANCH:      type/TICKET-description (feature/MEMO-123-x)
COMMIT:      type(scope): imperative (feat(auth): add PKCE)
```
