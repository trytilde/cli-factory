---
name: refactor-crate-to-flat-layout
description: Migrate a domain crate from the legacy `api/local/<domain>.rs` + `src/logic/<domain>.rs` split layout to the current flat layout (single `impl_local.rs`, types + freestanding fns colocated in `src/{domain}.rs`). Trigger when the user says "flatten this crate", "the layout looks old", or asks to apply the canonical structure to a crate that still has the split.
metadata:
  author: tilde
  version: "1.0.0"
  argument-hint: <crate-name>
---

# Refactor a crate to the flat layout

The current canonical layout is "flat" — a single `impl_local.rs`
file, with domain types **colocated** with their freestanding
business-logic functions in `src/{domain}.rs`. Older crates split
things across `api/local/<domain>.rs` + `src/logic/<domain>.rs`.
This skill migrates them.

Reference: `crates/inbox/`, `crates/skills/`, and
`crates/scheduled-jobs/` are the canonical post-refactor shapes.

## Before (legacy)

```
src/api/
  mod.rs              # trait
  local/
    mod.rs            # LocalFooApi struct + trait impl
    sessions.rs       # freestanding fns
    messages.rs       # freestanding fns
src/logic/
  sessions.rs         # types only
  messages.rs         # types only
```

## After (flat)

```
src/
  api/
    mod.rs            # FooApi trait
    impl_local.rs     # LocalFooApi struct + trait impl (single flat file)
  sessions.rs         # types AND freestanding fns
  messages.rs         # types AND freestanding fns
```

## Process

### 1 — Merge domain types and functions

For each domain (e.g. `sessions`, `messages`):

1. Move the freestanding fns from `src/api/local/{domain}.rs` into
   `src/{domain}.rs` next to the existing types.
2. Update imports: `crate::api::local::{domain}::` →
   `crate::{domain}::`, `crate::logic::{domain}::` →
   `crate::{domain}::`.

Each `src/{domain}.rs` should now contain BOTH types and functions.

### 2 — Flatten `impl_local`

1. Merge the contents of `src/api/local/mod.rs` (and any leftover
   helpers) into a single `src/api/impl_local.rs`.
2. Delete the `src/api/local/` directory.
3. In `src/api/mod.rs` change `pub mod local;` to
   `pub mod impl_local;`.
4. The trait impl in `impl_local.rs` should be thin: destructure the
   request and delegate.

```rust
async fn list_sessions(&self, request: ListSessionsRequest) -> Result<ListSessionsResponse, CommonError> {
    let ListSessionsRequest { pagination, inbox_id, org_id, team_id } = request;
    crate::session::list_sessions(&*self.repository, pagination, inbox_id, &org_id, &team_id).await
}
```

### 3 — Fix references workspace-wide

Search and replace:

- `{crate}::api::local::` → `{crate}::api::impl_local::`
- `{crate}::api::{domain}::` → `{crate}::{domain}::` (when types moved to root)
- `{crate}::logic::{domain}::` → `{crate}::{domain}::`

### 4 — Verify

```bash
cargo check -p {crate}
cargo check          # full workspace
cargo test  -p {crate}
```

## Reference pattern

Freestanding fn (`src/session.rs`):

```rust
pub async fn list_sessions(
    repo: &(impl SessionRepositoryLike + ?Sized),
    pagination: PaginationRequest,
    inbox_id: Option<String>,
    org_id: &str,
    team_id: &str,
) -> Result<ListSessionsResponse, CommonError> {
    // ...
}
```

## Key rules

- Domain modules contain BOTH types AND functions. No `src/logic/`
  directory.
- Trait methods are thin: destructure → delegate. No inline business
  logic.
- Multi-tenancy is via explicit `org_id` / `team_id` parameters, not a
  generic `Ctx` object — the OSS path pins them to the zero UUID.
- Request structs (e.g. `ListSessionsRequest`, `GetSessionRequest`)
  carry trait-method data params.
