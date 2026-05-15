---
name: writing-tests
description: Write meaningful tests in the tilde-agent Rust workspace. Use when adding/modifying tests in any `crates/*` crate. Establishes the integration-first philosophy, the mandatory `tilde_common::setup_test!()` macro, the `cargo nextest` runner, and the conventions for how tests load env, talk to the libsql/SQLite test DB, and exercise axum routers end-to-end.
---

# Writing tests for the tilde-agent workspace

## Run tests with nextest

Use **`cargo nextest run`**, not `cargo test`. Nextest is faster
(process-per-test isolation, parallel scheduling) and gives proper
per-test timing + retry support. Workspace config lives at
`.config/nextest.toml`.

```bash
# Whole workspace
make test                       # cargo nextest run --workspace
make test-doc                   # doctests (nextest doesn't run them yet)

# Targeted
cargo nextest run -p inbox --lib
cargo nextest run -p ai-gateway --test e2e_openai
cargo nextest run -E 'test(create_session)'   # filter expression

# CI profile (JUnit + retries on flake)
cargo nextest run --profile ci --workspace
```

**Doctests**: nextest doesn't run them. Use
`cargo test --workspace --doc` separately (also wired up as
`make test-doc`).

## Philosophy

**Integration and end-to-end tests over unit tests.** Prefer one test
that exercises a real call path (request → handler → repository →
sqlite → response) over five tests that mock everything in between.
Mocks lie; databases don't.

**Fewer, meaningful tests over many trivial ones.** A test that
validates the round-trip of a `CreateSession` request is worth more
than ten tests that each construct a different field of
`CreateSessionRequest`. From [AGENTS.md](../../AGENTS.md): "Never
write tests that just test construction of DTO-style objects."

**Tests must never be skippable.** From AGENTS.md: "Never write tests
that are skippable. If they fail due to missing env vars, they must
fail loudly." No `#[ignore]`, no `RUN_X=1` guards, no
early-return-on-missing-env. If a test requires a particular env var
and it's missing, **panic with a clear message** so CI / the dev sees
the gap.

## The `setup_test!()` macro is mandatory

Every test in this workspace **must** start with
`tilde_common::setup_test!()`.

What it does (idempotent across test threads via a `Once`):

1. **Configures the rustls crypto provider.** Without this, anything
   touching TLS (reqwest, axum HTTPS) panics.
2. **Loads `.env` and `.env.secrets`** via
   `tilde_common::env::load_optional_env_files()` — walks up from
   `CARGO_MANIFEST_DIR` to find them. Means **`cargo test` works
   without `source .env`**.
3. **Configures tracing** via
   `tilde_common::logging::configure_logging()` — gives you the same
   span/event output the binaries produce, so test failures come with
   structured logs.
4. **Sets up a per-test `DATA_DIR`** so tests that touch the
   filesystem don't collide.
5. (Optional) Sets a libsql connection string env var when called as
   `setup_test!(db_conn_string_key: "MY_LIBSQL_URL")`.

```rust
#[tokio::test]
async fn my_integration_test() {
    let ctx = tilde_common::setup_test!(); // returns TestContext { workspace_root, crate_root }
    // ... rest of the test reads env / opens connections / etc.
}
```

## libsql / SQLite-backed tests

The OSS workspace uses libsql/SQLite for everything. Test DBs live
under the per-test `DATA_DIR` from `setup_test!()`, so collisions
between parallel tests are impossible.

Repository tests instantiate the SQLite repo against an in-memory or
temp-dir libsql connection, run the migrations, then exercise the
trait. Reference patterns live in the existing crates — search for
`setup_test_libsql` / `tilde_common::test_utils` for the helper
functions.

## Integration tests at `<crate>/tests/`

Anything that exercises a router, gateway, or external API goes in
`<crate>/tests/<name>.rs`. Each file is its own binary; cargo will run
them in parallel by default — use `--test-threads=1` only when state
is shared.

Set up the server inline:

```rust
async fn start_test_server() -> String {
    let _ctx = tilde_common::setup_test!();

    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(listener, my_router()).await.unwrap();
    });
    format!("http://{addr}")
}
```

Patterns to copy:

- **Stub trait impls live in the test file** (or are re-exported from
  a sibling test file in the same crate).
- **Always-allow stubs are fine** for tests where the trait is on the
  call path but not the assertion target.
- **Bind ephemeral ports** (`127.0.0.1:0`) and read the assigned
  address back. Hard-coded ports cause flakes.
- **Spawn tracing on the same runtime** — `setup_test!()` configures
  the global subscriber so axum spans land in the test output.

## What not to do

| Anti-pattern | Why it's wrong | Correct approach |
|---|---|---|
| `#[test] fn dto_constructs() { let _ = Foo { ... }; }` | Tests no behaviour. AGENTS.md forbids. | Delete. If you want a smoke check, write one round-trip integration test that uses `Foo` for real. |
| `if env::var("X").is_err() { return; }` | Skippable test masks missing env in CI. | `let x = env::var("X").expect("X must be set in .env.secrets");` |
| `#[ignore = "needs Y"]` + `RUN_Y=1 cargo test --ignored` | Same as above — invisible failure mode. | Drop the `#[ignore]`. Fail loud. |
| Mocking the database in repository tests | The mock and the real schema diverge over time and bugs leak to prod. | Use a real libsql connection against the per-test `DATA_DIR`. |
| Hardcoded ports (`127.0.0.1:8080`) in tests | Port collisions = flake. | Bind `127.0.0.1:0`, read the assigned port, hand it back. |
| One assertion per test, ten tests per behaviour | Shotgun coverage that grows with the codebase. | One test per **behaviour** with multiple assertions if needed. |
| Skipping `setup_test!()` to "save time" | Test runs without TLS / env / tracing → confusing failures later. | Always include `setup_test!()` first. It's idempotent. |

## Adding tests for a new feature — checklist

1. **Pick the layer.** Repository (libsql-backed), trait (freestanding
   fn + DI), router (axum integration), or end-to-end
   (`<crate>/tests/`).
2. **Start with `tilde_common::setup_test!()`** as the first line.
3. **Use real impls where you can.** Real libsql connection for DB;
   spin up the actual axum router for HTTP.
4. **Stub only at the boundary** of the system under test. Never
   mid-stack.
5. **Generate unique IDs** so tests can run in parallel where the
   underlying schema allows it.
6. **Assert on behaviour, not internals.** "User can fetch the
   session they just created" beats "the SQL `INSERT` ran with these
   exact bytes".
7. **One round-trip > five field-level tests.**

## Repo references

- [AGENTS.md](../../AGENTS.md) — "Tests" section + the test-related
  bullet points
- [`.config/nextest.toml`](../../.config/nextest.toml) — nextest
  workspace config (default + ci profiles)
- `crates/common/src/test_utils/helpers.rs` — `setup_test!()` macro
