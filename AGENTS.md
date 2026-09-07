# CLI Factory Agent Guide

CLI Factory is an agent-first Go CLI for discovering and invoking curated SaaS/tool provider commands.

## Core Rules

- Prefer curated high-level **Tools** over low-level CRUD wrappers.
- Provider code lives under `providers/<provider>`.
- Tool code lives under `providers/<provider>/<tool>`.
- Provider and tool metadata drive search, discovery, generated docs, and generated catalogues.
- Credentials are not stored. Auth and connection details are optional provider parameters passed per invocation.
- Commands print `SUCCESS` or `FAILURE` by default and write full JSON logs to the local invocation log cache.
- Use `--debug` when a full result or error object should be printed directly.

## Provider Workflows

- Use `skills/add-provider/SKILL.md` for a new provider.
- Use `skills/update-provider/SKILL.md` for existing providers and tools.
- Run e2e tests for provider behavior; keep unit tests focused on internal framework packages.
- Do not run destructive provider e2e tests until the user confirms target account/workspace/project, cleanup expectations, and spend/rate-limit constraints.
- Root `secrets.yaml` is plaintext and gitignored. Use `make sops-encrypt` / `make sops-decrypt` to manage `secrets.enc.yaml` and `.env.secrets`.
- `OPENAI_API_KEY` for `make generate-catalog` can come from `.env.secrets`.
- Catalogue embeddings are embedded as `catalog/embeddings.bin`, a compact float32 binary index. The JSON file is only the manifest.
- Human docs are generated into the `public-docs/` git submodule, which points at the shared `trytilde/docs` Mintlify repo. `make generate-docs` initializes the submodule before writing generated pages.

## Required Checks

```bash
make test-unit
make test-e2e
make generate-docs
make generate-catalog
make build
```

Use `make build-all` before release-sensitive changes.

## Required documentation maintenance

For every change, follow [docs/README.md](docs/README.md). Read relevant
`docs/adrs/` first; create or update records for durable decisions and keep setup,
README, and public documentation in sync. Maintain the complete change record at
`docs/updates/pending/<short-slug>.md` until a PR number exists, then rename it to
`docs/updates/<actual-pr-number>.md` and refresh it after every revision. Missing
or stale required documentation blocks completion. Record already authorized
decisions directly; ask only about unresolved choices. See
[maintain-docs](.agents/skills/maintain-docs/SKILL.md) for the workflow.
