---
name: genesis-auth
description: Use when adding authentication to a project scaffolded with genesis — login, sessions, SSO, admin users. Asks for an auth tier (basic/standard/strict), derives the mechanism from requirements, and implements it in place following the project's existing conventions.
user-invocable: true
disable-model-invocation: true
argument-hint: "[basic | standard | strict] [google | apple]..."
---

# genesis-auth

Adds authentication to a genesis project. The user picks a tier and states requirements; the skill derives the mechanism — never quiz the user on implementation choices like JWT vs cookie — then writes the code directly into the target project.

This skill is a guideline, not a copy job. There is no reference implementation to clone: read the target project's existing code and write auth that matches it.

## Tiers

| Tier | Use case | Includes |
|---|---|---|
| `basic` | Internal tools | Single admin seeded from env var, password login, cookie session in Postgres. No signup, no email. |
| `standard` | Normal apps | Email+password users, optional Google/Apple SSO, sessions in Postgres, password reset, admin/user roles. |
| `strict` | Sensitive data | `standard` + TOTP MFA, rate limiting + lockout, session rotation on privilege change, audit log. Beyond this, recommend an external IdP (Keycloak, Auth0, WorkOS) instead of more homegrown code. |

## Question flow

1. **Read the arguments first.** `basic`/`standard`/`strict` answers the tier; `google`/`apple` answers SSO. Ask only what is still missing.
2. **Ask remaining questions in ONE AskUserQuestion call**, with the recommended option first and marked "(Recommended)":
   - Tier: `basic` | `standard (Recommended)` | `strict`
   - Clients: `browser only (Recommended)` | `also mobile/API consumers`
   - SSO providers (multiSelect): `google` | `apple` — an empty selection means no SSO. Omit this question only when the tier is already known to be `basic`; if the tier is asked in the same call and the user picks `basic`, ignore the SSO answer.
3. **Derive the mechanism — do not ask about it:**
   - browser only → cookie (HttpOnly, Secure, SameSite=Lax) + server-side session token in Postgres
   - mobile/API consumers → same cookie flow for the web, plus a token path (JWT access + refresh, or API keys for machine consumers)
4. **Confirm with the derived plan in prose, not more pickers.** Example: "basic tier: seeded admin, cookie sessions in Postgres, no SSO — proceed?" If the user overrides a mechanism (e.g. "actually JWT"), honor it.

## Implementation rules

1. **Read the target project before writing anything.** Its router, config struct, migration numbering, and package layout decide the shape of the new code. Match them; do not impose a layout from this document.
2. **Extend the existing users table, don't invent a parallel one.** A genesis server already ships `internal/db/migrations/0001_create_users.sql` with `password_hash` and a `user_role` enum. Add columns and new tables (sessions, password resets, SSO identities, MFA secrets) in a new numbered migration.
3. **Follow the established layout** for a genesis server:
   - `internal/auth/` — password hashing, session/token issue and verify, provider clients. No gin types in here.
   - `internal/api/http/handler/auth.go` — login, logout, signup, reset endpoints.
   - `internal/api/http/dto/auth.go` — request/response structs.
   - `internal/api/http/middleware/auth.go` — session/bearer verification, role checks.
   - `internal/db/migrations/NNNN_*.sql` — goose `-- +goose Up` / `Down` blocks, both directions.
   - `internal/db/queries/*.sql` — sqlc named queries; run `sqlc generate` after editing.
4. **Wire everything.** Register handlers and middleware in `SetupRoute`, add any new dependency to the `Services` struct, and construct it in `main.go`. Auth code that compiles but is never mounted looks done and isn't.
5. **Config goes through the existing mechanism.** Add fields to the config struct in `cmd/<app>/config.go`, defaults to `application.yml`, and every new key to `.env.example` using the `.` → `_` upper-case form (`auth.session_ttl` → `AUTH_SESSION_TTL`). Never read `os.Getenv` directly.
6. **Use the standard library and existing deps first.** `golang.org/x/crypto/bcrypt` for passwords, `crypto/rand` for tokens. Add a dependency only when the tier genuinely needs it (JWT, TOTP), and run `go mod tidy`.
7. **Never log or return secrets.** No password, hash, session token, or reset token in logs or error responses.

## Verification procedure

1. `make build` passes in the target server.
2. `sqlc generate` is clean and the generated code is committed.
3. A migration file exists — up *and* down — for every new table or column the code queries.
4. Every new config key the code reads appears in `.env.example` and `application.yml`.
5. For cookie sessions: the cookie is set with `HttpOnly`, `Secure`, and `SameSite`.
6. Session tokens are stored hashed, with an expiry column, and logout deletes the row.

## Common mistakes to watch for

- **Asking mechanism questions.** Never ask "JWT or cookie?" — derive it from the Clients answer.
- **Re-asking what arguments already answered.** `/genesis-auth basic` should ask nothing except the final confirmation.
- **Copying without wiring.** Routes and middleware that are never mounted on the router.
- **A second users table.** Extend the one the project already has.
- **Migrations without a `Down` block**, or editing an already-applied migration instead of adding a new one.
- **Hand-editing `internal/db/sqlc/`.** It is generated; change the `.sql` query and regenerate.
