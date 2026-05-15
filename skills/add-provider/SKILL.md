---
name: add-provider
description: Create a brand-new cli-factory provider and its first curated agent-facing command set, including metadata, schemas, prompts, implementation, SOPS-backed e2e secrets, and iterative e2e validation.
---

# Add Provider

Use this skill when creating a new provider under `providers/<provider>` or adding the provider's first tool set. The goal is not to mirror an API surface. The goal is to create a small set of high-value **Tools** that agents can safely discover, inspect, and invoke.

## Operating Rules

- Explore repo files before asking questions that the repo can answer.
- Ask one high-impact question at a time. Do not ask a batch of broad questions until after source gathering.
- Challenge fuzzy terms and overloaded provider concepts. If "account", "workspace", "project", "tenant", or "user" could mean more than one remote object, stop and resolve the vocabulary.
- Use concrete scenarios to force tool boundaries: "Should this command create a draft, send immediately, or support both?" is better than "What email commands do you want?"
- Resolve provider/tool boundaries before implementation.
- Propose `CONTEXT.md` glossary additions and ADRs when decisions crystallize. Keep glossary entries domain-only; put implementation decisions in ADRs.
- Prefer non-destructive tests and dry-run modes. Before running e2e, confirm params so nothing destructive happens.

## Phase 1: Ground In The Repo

1. Inspect existing provider layout:
   - `find providers -maxdepth 4 -type f`
   - `find skills -maxdepth 3 -type f`
   - `find docs -maxdepth 3 -type f`
2. Read the current metadata/schema conventions if they exist.
3. Inspect the test harness and Make targets:
   - `internal/testharness`
   - `Makefile`
4. Inspect shared Mintlify docs conventions:
   - `docs/` git submodule
   - `DOCS_REPO_DIR`
   - `<docs repo>/docs.json`
   - `internal/docgen`
   - `tools/generatedocs`
5. Check `.gitignore` includes:
   - `.tmp/`
   - `providers/**/test_secrets.yaml`
   - `providers/**/override_test_secrets.yaml`
6. If the repo disagrees with these instructions, surface the conflict before editing.

## Phase 2: Gather External References

When the user provides docs, OpenAPI, GraphQL, example webpages, OSS clients, SDKs, or API contracts, gather them before planning.

Allowed actions:

- `git clone` OSS clients or SDKs into `.tmp/provider-references/<provider>/`.
- Download OpenAPI, GraphQL, JSON Schema, or other API contract files into `.tmp/provider-references/<provider>/`.
- Browse official docs and examples.
- Inspect existing client abstractions, auth setup, pagination, retries, and domain naming.

After gathering references, summarize:

- useful official APIs
- authentication and provider parameter patterns
- rate limits, quotas, and retry semantics
- destructive operations and remote resources they affect
- candidate high-level workflows
- SDK/client abstractions worth borrowing conceptually
- gaps, contradictions, and unclear terms

Then ask a concise follow-up question list, but still ask one high-impact question at a time while resolving answers.

## Phase 3: Grill The Provider Design

Integrate the grill-with-docs process directly:

- Ask one high-impact question at a time and provide your recommended answer.
- If a question can be answered by docs or code, inspect first.
- Challenge glossary conflicts immediately.
- Sharpen vague language into canonical terms.
- Use concrete scenarios, especially around destructive writes, identity, workspace/account boundaries, pagination, and idempotency.
- Propose `CONTEXT.md` terms for provider-specific language once resolved.
- Offer ADRs only when a decision is hard to reverse, surprising without context, and the result of a real trade-off.

Resolve:

- provider id, display name, aliases, categories
- `short_description` and `long_description`
- provider-level optional params such as `bearer_token`, `api_key`, `base_url`, account/workspace/project ids, and dry-run flags
- curated tool list and command paths
- destructive behavior, dry-run support, idempotency keys, and cleanup strategy
- search aliases and example user intents
- e2e test target account/workspace/project

Reject raw CRUD dumps unless the user explicitly justifies a low-level primitive.

## Phase 4: Plan Files

For each new provider, create:

```text
providers/<provider>/
├── mod.go
├── cli-metadata.yaml
├── generator-metadata.yaml
├── generator-prompt.md
├── test_secrets.example.yaml
├── test_secrets.enc.yaml
└── <tool>/
    ├── mod.go
    ├── cli-metadata.yaml
    ├── generator-prompt.md
    ├── input-schema.yaml
    ├── output-schema.yaml
    └── e2e_test.go
```

Use hyphenated command directories, for example `send-email`, because the directory mirrors the command path. Use valid Go package names inside files.

Mintlify human documentation is generated from these metadata and schema files into the `docs/` git submodule for the shared `trytilde/docs` repo:

```text
<docs repo>/projects/cli-factory/providers/<provider>/index.mdx
<docs repo>/projects/cli-factory/providers/<provider>/tools/<tool>.mdx
```

The generated provider page must summarize the provider, categories, aliases, provider parameters, and tools. The generated tool page must summarize what the tool does, command path, provider parameters, input params, output fields, full input schema, and full output schema.

## Phase 5: Test Secret Handshake

Before implementation/test loops, ask the user to create provider secrets.

Use this exact operational instruction:

```text
Create providers/<provider>/test_secrets.yaml from providers/<provider>/test_secrets.example.yaml, populate it with real test credentials, then tell me when it is ready. If these credentials are temporary or experimental, create providers/<provider>/override_test_secrets.yaml instead.
```

Rules:

- `test_secrets.example.yaml` is committed.
- `test_secrets.yaml` is plaintext, gitignored, and used as the source for SOPS encryption.
- `test_secrets.enc.yaml` is committed.
- `override_test_secrets.yaml` is always gitignored, never encrypted, never committed, and always overrides other secret sources for local e2e runs.
- Use the tilde SOPS KMS key:

```text
arn:aws:kms:us-east-1:914788356809:alias/tilde-app-dev-sops
```

After the user confirms:

1. If using `test_secrets.yaml`, run:

   ```text
   make sops-encrypt-provider-test-secrets PROVIDER=<provider>
   ```

2. If using `override_test_secrets.yaml`, do not encrypt it.
3. Run targeted e2e tests.

## Phase 6: Confirm E2E Safety Before Running Tests

Before running e2e tests, ask the user to confirm:

- exact provider and tool tests to run
- whether tests may perform writes or destructive operations
- target account/workspace/project ids
- whether to use `override_test_secrets.yaml`
- cleanup expectations for created remote resources
- max spend, quota, and rate-limit constraints
- whether dry-run/non-destructive mode is required

Default to non-destructive e2e tests when possible.

## Phase 7: Implement And Iterate

Use this loop:

1. Make the smallest provider/tool implementation change.
2. Run the targeted e2e command:

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

Run each new command's e2e test before broad provider tests. Then run:

```text
make test-provider PROVIDER=<provider>
```

After metadata and schemas are stable, regenerate human docs:

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

- Metadata validates.
- Schemas validate.
- Mintlify docs generated with `make generate-docs`.
- `<docs repo>/projects/cli-factory/providers/<provider>/index.mdx` summarizes the provider from metadata.
- `<docs repo>/projects/cli-factory/providers/<provider>/tools/<tool>.mdx` includes the tool summary, params, input schema, and output schema.
- `<docs repo>/docs.json` navigation includes the CLI Factory project, provider, and tool pages.
- Catalogue generated.
- Embeddings regenerated.
- Each new provider command has an e2e test.
- Provider e2e tests pass or are explicitly blocked with the external reason.
- Invocation logs inspected for failed attempts.
- Secrets are redacted from logs.
- `providers/<provider>/test_secrets.yaml` is gitignored.
- `providers/<provider>/override_test_secrets.yaml` is gitignored.
- `providers/<provider>/test_secrets.enc.yaml` exists when shared provider e2e secrets are required.
- No low-level CRUD commands were added without explicit justification.
