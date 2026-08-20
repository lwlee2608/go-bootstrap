---
name: genesis-auth
description: Use when adding authentication to a project scaffolded with genesis — login, sessions, SSO, admin users. Asks for an auth tier (basic/standard/strict), derives the mechanism from requirements, and scaffolds from the reference implementation in github.com/lwlee2608/genesis.
user-invocable: true
disable-model-invocation: true
argument-hint: [basic | standard | strict] [google | apple]...
---

# genesis-auth

Adds authentication to a genesis project by copying from the auth reference in [`lwlee2608/genesis`](https://github.com/lwlee2608/genesis). The user picks a tier and states requirements; the skill derives the mechanism — never quiz the user on implementation choices like JWT vs cookie.

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

## Scaffolding rules

1. **Clone the live repo, never work from memory.** `git clone --depth=1 https://github.com/lwlee2608/genesis /tmp/genesis-template`, then read the auth reference under `/tmp/genesis-template/reference/project-00/services/project-00-server/internal/auth/<tier>/`. The repo evolves; reproducing contents from memory causes drift.
2. **If the reference for the chosen tier does not exist yet, stop and say so.** List which tiers exist in the clone. Do not improvise an implementation from memory — that defeats the point of the reference.
3. **Adapt names, keep structure.** Replace `project-00` with the target project's name everywhere (module path, directories, Makefile `APP`), keeping the underscore variant for Postgres identifiers — same rules as the `genesis` skill.
4. **Wire, don't just copy.** After copying: register the auth routes/middleware in the router, add the migration files under `internal/db/migrations`, add new env vars to `.env.example`, and run `sqlc generate` if queries were added.

## Verification procedure

1. `make build` passes in the target server.
2. A migration file exists for every new table the copied code queries.
3. Every new env var the code reads appears in `.env.example`.
4. For cookie sessions: the session cookie is set with `HttpOnly` and `SameSite`.

## Common mistakes to watch for

- **Asking mechanism questions.** Never ask "JWT or cookie?" — derive it from the Clients answer.
- **Re-asking what arguments already answered.** `/genesis-auth basic` should ask nothing except the final confirmation.
- **Improvising a missing tier.** If `reference/.../auth/strict` is absent, report it — do not write it from scratch.
- **Copying without wiring.** Auth code that compiles but is never mounted on the router looks done and isn't.
