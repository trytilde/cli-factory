<p align="center">
  <a href="https://github.com/trytilde/cli-factory"><img src="https://img.shields.io/badge/GitHub-trytilde%2Fcli--factory-181717?style=for-the-badge&logo=github" alt="GitHub"></a>
  <a href="https://discord.gg/jj7sNyCGD4"><img src="https://img.shields.io/badge/Discord-Join%20the%20community-5865F2?style=for-the-badge&logo=discord&logoColor=white" alt="Join our Discord"></a>
</p>

# CLI Factory

CLI Factory builds `factory`: a single static binary for AI agents to discover and invoke curated SaaS/tool commands.

Most CLIs are human-first. `factory` is agent-first: semantic search, progressive discovery, structured schemas, terse status output, full JSON invocation logs, and provider commands designed as useful workflows rather than raw API CRUD.

## Why this exists

Agents need tool access that is compact, predictable, and safe to reason about in limited context windows. CLI Factory provides a repeatable way to add SaaS providers with high-level commands like:

```bash
factory google send-email --to user@example.com --subject "Hello" --body "..."
```

The CLI helps agents find the right capability, inspect only the detail they need, invoke it once with explicit params, and keep full output in local logs instead of flooding the conversation.

## Features

- Single static Go binary for macOS, Linux, and Windows
- Agent-first semantic search across providers and tools
- Progressive discovery with `factory discover short` and `factory discover long`
- Default output is only `SUCCESS` or `FAILURE` plus a full JSON log path
- `--debug` for full stdout/stderr JSON when needed
- Provider-level optional auth/connection params such as `--bearer-token`, `--api-key`, and `--base-url`
- No credential storage or profiles
- Compact embedded binary embedding catalogue
- Provider e2e test harness with SOPS-encrypted `test_secrets.enc.yaml`
- Generated provider docs into the shared `trytilde/docs` Mintlify submodule
- Add/update provider skills for contributors

## Install

Latest release:

```bash
curl -sSfL https://raw.githubusercontent.com/trytilde/cli-factory/main/install.sh | bash
```

This installs `factory` to `~/.factory/bin/factory`.

Install a specific release:

```bash
curl -sSfL https://raw.githubusercontent.com/trytilde/cli-factory/main/install.sh | bash -s -- --version v0.1.0
```

Add it to your shell path if needed:

```bash
export PATH="$HOME/.factory/bin:$PATH"
```

## Quickstart

Search first:

```bash
factory search "send an email"
```

Then progressively discover only what you need:

```bash
factory discover short google
factory discover short google send-email
factory discover long google send-email
```

Invoke directly:

```bash
factory google send-email --bearer-token "$TOKEN" --to user@example.com --subject "Hello" --body "Hi"
```

By default, commands print:

```text
SUCCESS
full logs at /absolute/path/to/log.json
```

Use `--debug` when you want the full JSON result in the terminal.

## Development

Clone with the docs submodule:

```bash
git clone --recurse-submodules https://github.com/trytilde/cli-factory
cd cli-factory
```

Run the core checks:

```bash
make generate-metadata
go test ./...
make generate-docs
make generate-catalog
make build
```

Cross-compile static binaries:

```bash
make build-all
```

CI runs unit tests, provider e2e tests, docs/catalogue generation, and static builds. Releases publish Go binaries to GitHub Releases.

## Add or update a provider

Use the repo skills:

- `skills/add-provider/SKILL.md` for a new provider
- `skills/update-provider/SKILL.md` for existing provider changes
- `skills/use-factory-cli/SKILL.md` for production agent usage

Provider layout:

```text
providers/<provider>/
├── cli-metadata.yaml
├── metadata_gen.go
├── generator-metadata.yaml
├── generator-prompt.md
├── test_secrets.example.yaml
├── test_secrets.enc.yaml
├── mod.go
└── <tool>/
    ├── cli-metadata.yaml
    ├── input-schema.yaml
    ├── output-schema.yaml
    ├── metadata_gen.go
    ├── mod.go
    └── e2e_test.go
```

Provider rules:

- Build high-level agent workflows, not low-level CRUD dumps.
- Treat `cli-metadata.yaml`, `input-schema.yaml`, and `output-schema.yaml` as the source of truth for provider/tool metadata and schemas.
- Run `make generate-metadata` after metadata/schema edits. It emits `metadata_gen.go` files with static strings and schema maps compiled into the `factory` binary.
- Do not hand-edit `metadata_gen.go`, or duplicate descriptions, categories, aliases, provider params, or JSON schemas in handwritten Go files.
- For OAuth client flows, the CLI should usually accept an access token or bearer token provider parameter. E2E tests may use durable test secrets such as `client_id`, `client_secret`, and `refresh_token` to mint a fresh access token before invoking the CLI.
- Add/update e2e tests for every provider command.
- Use `override_test_secrets.yaml` for local throwaway credentials; never commit it.
- Commit only encrypted shared provider secrets as `test_secrets.enc.yaml`.
- Run `make generate-docs` and `make generate-catalog` after metadata/schema changes.

## Harness-driven provider work

`harness-shop` is included as a submodule for structured provider-building workflows. The Factory CLI provider harness is separate from the generic experiment harness, but reuses shared UI pieces such as chat, runs, diffs, and secrets.

Factory provider harness runs should follow these phases:

- Discovery: confirm provider goals, tool goals, auth model, docs, examples, and reference links.
- Plan: define provider behavior, each tool, and input/output schemas before implementation.
- Testing: define real Go e2e tests, required credentials, cleanup behavior, target account/workspace, and rate/spend limits.
- Implementation: clone CLI Factory into a run-specific checkout, create a `provider-harness/<id>` branch, write provider code and Go e2e tests, then iterate until targeted e2e tests and required checks pass.

The harness secrets form writes `providers/<provider>/override_test_secrets.yaml` for local runs. Shared secrets should be promoted to `test_secrets.yaml`, encrypted with SOPS, and committed as `test_secrets.enc.yaml`.

## Contribute

CLI Factory is intended to become a shared catalogue of high-quality agent tools. If you want agents to use your SaaS product well, contribute a provider with a small set of thoughtful commands and real e2e tests.

Star the project, open issues or PRs, and join the Tilde community on [Discord](https://discord.gg/jj7sNyCGD4).

## Documentation locations

Repository decisions and change records live in [docs/](docs/README.md). Shared
Mintlify pages are generated into the independent `public-docs/` submodule. Run
`git submodule sync` and `git submodule update --init --recursive public-docs`
after updating an existing checkout; `make generate-docs` initializes it as needed.
`DOCS_REPO_DIR` can select a different checkout of the shared documentation.
