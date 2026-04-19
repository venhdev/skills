# naming-conventions

> **Naming is the most permanent decision in a codebase.** A bad name survives refactors, team changes, and cloud migrations.

A Claude Code skill that ensures every new identifier — service, class, file, function, env var, DB column, gRPC proto, and more — is **unambiguous, future-safe, and consistent** with industry standards.

---

## What it does

Applies a decision tree per naming layer and a universal anti-pattern checklist to every new identifier. Prevents the most common and costly naming mistakes before they happen:

- Encoding tech stack, team, region, or environment into names
- I-prefix on TypeScript interfaces (redundant in structural types)
- Mixed plural/singular, wrong casing per layer
- Non-standard service, file, function, DB, gRPC, and flag names

## When it triggers

Use this skill whenever naming:

- A new **service**, **module**, or **feature**
- A **class**, **interface**, **type**, or **enum**
- A **file** or **directory**
- A **function** or **method**
- An **environment variable**
- A **database table** or **column**
- A **gRPC proto** package, message, or method
- A **feature flag** or **git branch/commit**

Also triggers proactively when a new name could collide with existing ones.

---

## Layers covered

| # | Layer | Example |
|---|---|---|
| 1 | Service & Architecture | `auth-service`, `graphql-bff`, `api-gateway` |
| 2 | Class / Interface / Type / Enum | `UserService`, `CreateUserDto`, `OrderStatus` |
| 3 | File & Directory | `UserService.ts`, `create-user.dto.ts` |
| 4 | Function & Method | `getUserById`, `isAuthenticated`, `onUserCreated` |
| 5 | JSON Keys & API Payloads | `userId`, `lineItems`, `createdAt` |
| 6 | Environment Variables | `NOTIFICATION_RABBITMQ_URL` |
| 7 | Database Schema | `users` (table), `user_id` (column) |
| 8 | gRPC / Protobuf | `myapp.notification.v1`, `GetUser` |
| 9 | Feature Flags | `feature-new-checkout-flow` |
| 10 | Git Branch & Commit | `feature/MEMO-123-x`, `feat(auth): add PKCE` |

---

## Install

### Option 1 — Direct path (recommended for personal use)

```bash
npx skills add https://github.com/yourusername/vskills --skill naming-conventions
```

### Option 2 — Dedicated repo (for sharing / marketplace)

```bash
npx skills add yourusername/agent-skills --skill naming-conventions
```

See [skills.sh](https://skills.sh) for full CLI documentation.

---

## File structure

```
naming-conventions/
├── SKILL.md          ← the skill (required)
├── icon.svg          ← marketplace icon
└── README.md         ← this file
```

---

## Design principles

- **Name the concept, not the implementation.** `payment-service` not `stripe-service`.
- **Tech stack independence.** Names survive a rewrite.
- **Consistency over cleverness.** Every name in the same category follows the same pattern.
- **Anti-patterns over rules.** Instead of prescribing every correct name, the skill eliminates the most costly mistakes first.

---

## Quick reference

| Category | Convention | Example |
|---|---|---|
| Services | `kebab-case` | `auth-service` |
| Classes | `PascalCase`, no I-prefix | `UserService` |
| Files (class) | `PascalCase.ts` | `UserService.ts` |
| Files (DTO/util) | `kebab-case.<kind>.ts` | `create-user.dto.ts` |
| Functions | `camelCase`, verb prefix | `getUserById()` |
| Env vars | `UPPER_SNAKE_CASE` | `AUTH_JWT_SECRET` |
| DB tables | `snake_case`, plural | `users` |
| DB columns | `snake_case`, singular | `user_id` |
| Proto pkg | `lowercase.v1` | `myapp.auth.v1` |
| Flags | `kebab-case` | `feature-checkout-v2` |
