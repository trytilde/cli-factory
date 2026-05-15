---
name: update-provider
description: Update an existing cli-factory provider or tool while preserving compatibility, refreshing references, updating e2e tests, and iterating against targeted provider e2e runs.
---

# Update Provider

Use this skill when changing an existing provider, adding a tool to an existing provider, updating schemas/metadata, or fixing provider behavior. The default posture is conservative: preserve existing metadata and backwards compatibility unless the user explicitly approves a breaking change.

## Operating Rules

- Explore repo files before asking discoverable questions.
- ask one high-impact question at a time. Provide your recommended answer.
- Challenge fuzzy terms and overloaded provider concepts.
- Use concrete scenarios to test boundaries and failure modes.
- Resolve compatibility, tool boundaries, and e2e safety before implementation.
- Do not rename provider ids, tool ids, command paths, parameters, output fields, aliases, or categories unless explicitly approved.
- Preserve existing metadata by default; only change descriptions, aliases, categories, schemas, or prompts when the update requires it.
- Default to non-destructive e2e tests.

## Phase 1: Inspect Existing Provider

Read:

- `providers/<provider>/mod.go`
- `providers/<provider>/cli-metadata.yaml`
- `providers/<provider>/generator-metadata.yaml`
- `providers/<provider>/generator-prompt.md`
- affected tool directories
- affected `input-schema.yaml` and `output-schema.yaml`
- affected `e2e_test.go`
- generated Mintlify docs under `docs/projects/cli-factory/providers/<provider>/`
- recent invocation logs if the user provided a failing command output

Also inspect:

- `internal/testharness`
- `internal/docgen`
- `Makefile`
- `.gitignore`
- shared docs submodule `docs/docs.json`
- relevant `CONTEXT.md` terms and ADRs

If the repo already has behavior that conflicts with the requested update, surface it before editing.

## Phase 2: Gather Updated References

When the user provides updated docs, OpenAPI, GraphQL, OSS clients, changelogs, SDKs, issue links, or examples, gather them before planning.

Allowed actions:

- `git clone` OSS references into `.tmp/provider-references/<provider>/`.
- Download API contracts into `.tmp/provider-references/<provider>/`.
- Browse official docs, migration guides, changelogs, and examples.
- Inspect SDK/client changes for auth, pagination, retries, and response shape changes.

Summarize:

- what changed in the external API
- affected commands and schemas
- compatibility risks
- new or changed destructive operations
- test secret changes required
- source references used
- gaps or contradictions

Then ask only the remaining high-impact questions.

## Phase 3: Grill The Update

Integrate the grill-with-docs process directly:

- Ask one high-impact question at a time.
- Inspect code/docs instead of asking answerable questions.
- Challenge glossary conflicts immediately.
- Sharpen vague terms into canonical language.
- Use concrete scenarios for old vs new behavior.
- Propose `CONTEXT.md` updates for resolved domain language.
- Offer ADRs only when a decision is hard to reverse, surprising, and trade-off driven.

Resolve:

- exact provider/tool scope
- whether this is additive, behavior-changing, or breaking
- whether old params/output fields remain supported
- e2e tests to update or add
- whether test secrets need new keys/scopes
- destructive behavior and cleanup requirements
- migration notes for agents

## Phase 4: Compatibility Rules

- Preserve existing metadata unless changed intentionally.
- Preserve backwards compatibility by default.
- Do not rename provider ids, tool ids, command paths, params, or outputs without explicit approval.
- If a param must change, prefer adding a new param and deprecating the old one in metadata.
- If output must change, preserve existing fields where possible and add new fields.
- If behavior changes, update `long_description`, examples, and e2e tests in the same change.
- If provider/tool metadata or schemas change, regenerate Mintlify docs in the same change.
- Generated docs are derived from metadata and schemas; do not hand-edit generated provider/tool docs to hide stale metadata.

## Phase 5: Test Secret Handling

Provider secret files:

```text
providers/<provider>/test_secrets.example.yaml
providers/<provider>/test_secrets.yaml
providers/<provider>/test_secrets.enc.yaml
providers/<provider>/override_test_secrets.yaml
```

Rules:

- `override_test_secrets.yaml` always overrides `test_secrets.yaml` and decrypted `test_secrets.enc.yaml` for local e2e runs.
- `override_test_secrets.yaml` is never encrypted and never committed.
- `test_secrets.yaml` is plaintext, gitignored, and used as the source for SOPS encryption.
- `test_secrets.enc.yaml` is committed.
- Use the tilde SOPS KMS key:

```text
arn:aws:kms:us-east-1:914788356809:alias/tilde-app-dev-sops
```

If the update needs new or changed secrets, ask the user:

```text
Update providers/<provider>/test_secrets.yaml from providers/<provider>/test_secrets.example.yaml, populate the new values, then tell me when it is ready. If these credentials are temporary or experimental, put them in providers/<provider>/override_test_secrets.yaml instead.
```

After confirmation:

1. If using `test_secrets.yaml`, run:

   ```text
   make sops-encrypt-provider-test-secrets PROVIDER=<provider>
   ```

2. If using `override_test_secrets.yaml`, do not encrypt it.
3. Continue with targeted e2e tests.

## Phase 6: Confirm E2E Safety Before Running Tests

Before running e2e tests, ask the user to confirm:

- exact provider and tool tests to run
- whether tests may perform writes or destructive operations
- target account/workspace/project ids
- whether to use `override_test_secrets.yaml`
- cleanup expectations for created remote resources
- max spend, quota, and rate-limit constraints
- whether dry-run/non-destructive mode is required

Do not run destructive e2e tests until the user confirms the target and cleanup behavior.

## Phase 7: Implement And Iterate

Use this loop:

1. Make the smallest provider/tool implementation change.
2. Run the affected command e2e test first:

   ```text
   make test-provider-tool PROVIDER=<provider> TOOL=<tool>
   ```

   or:

   ```text
   go test ./providers/<provider>/<tool> -run TestE2E
   ```

3. Read the invocation log path from `SUCCESS` or `FAILURE` output.
4. Inspect the JSON log.
5. Fix implementation, schema, metadata, prompt, or test expectations.
6. Re-run the same targeted e2e.
7. Repeat until successful or blocked by external API/account state.

After affected command tests pass, run regression e2e tests for related commands, then:

```text
make test-provider PROVIDER=<provider>
```

If metadata or schemas changed, regenerate human docs:

```text
make generate-docs
```

Inspect:

```text
<docs repo>/projects/cli-factory/providers/<provider>/index.mdx
<docs repo>/projects/cli-factory/providers/<provider>/tools/<tool>.mdx
<docs repo>/docs.json
```

## Required Final Checklist

- Existing metadata preserved unless intentionally changed.
- Backwards compatibility preserved or explicit breaking-change approval recorded.
- Metadata validates.
- Schemas validate.
- Mintlify docs generated with `make generate-docs` when metadata or schemas changed.
- `<docs repo>/projects/cli-factory/providers/<provider>/index.mdx` reflects provider metadata.
- `<docs repo>/projects/cli-factory/providers/<provider>/tools/<tool>.mdx` reflects tool metadata, params, input schema, and output schema.
- `<docs repo>/docs.json` navigation includes any added provider/tool pages.
- Catalogue generated.
- Embeddings regenerated.
- Affected command e2e tests pass or are explicitly blocked with the external reason.
- Related regression e2e tests pass or are explicitly skipped with justification.
- Invocation logs inspected for failed attempts.
- Secrets are redacted from logs.
- `providers/<provider>/test_secrets.yaml` is gitignored.
- `providers/<provider>/override_test_secrets.yaml` is gitignored.
- `providers/<provider>/test_secrets.enc.yaml` exists when shared provider e2e secrets are required.
