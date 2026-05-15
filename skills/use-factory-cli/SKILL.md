---
name: use-factory-cli
description: Use the factory CLI from production agents with progressive semantic discovery, terse command output, invocation logs, and context-preserving workflows.
---

# Use Factory CLI

Use this skill when an AI agent needs to discover and invoke SaaS/tool capabilities through the `factory` CLI in production.

`factory` is agent-first. Do not browse the whole catalogue into context. Start with semantic search, progressively discover only the provider/tool you need, then invoke with explicit parameters.

## Core Principles

- Preserve context by searching first and discovering narrowly.
- Prefer `discover short` before `discover long`.
- Use `discover long` only for the provider or tool you are likely to invoke.
- Treat default command output as a status pointer, not the data payload.
- Read the JSON invocation log when you need full results.
- Use `--debug` only when you intentionally want full result/error JSON in the current context.
- Pass auth and connection information as invocation parameters; do not expect stored profiles.
- Prefer non-destructive or dry-run provider parameters when the tool supports them.

## Discovery Flow

Start with semantic search:

```bash
factory search "send an email to a customer"
```

Search returns ranked provider and tool results in the invocation log. Each result includes next-step commands such as:

```text
factory discover short google
factory discover long google
factory discover short google send-email
factory discover long google send-email
```

Use the shortest discovery command that can answer your immediate question.

## Short Discovery

Use short discovery to decide whether a provider or tool is relevant without loading schemas.

Provider:

```bash
factory discover short <provider>
```

Tool:

```bash
factory discover short <provider> <tool>
```

Short tool discovery returns:

- tool summary
- categories
- provider parameter names
- input parameter names
- output field names

It intentionally omits full JSON schemas to preserve context.

## Long Discovery

Use long discovery only when you are ready to invoke or need exact schema details.

Provider:

```bash
factory discover long <provider>
```

Tool:

```bash
factory discover long <provider> <tool>
```

Long tool discovery returns:

- long description
- command path
- aliases
- provider parameters
- full input schema
- full output schema

## Invocation

Prefer direct command paths when known:

```bash
factory <provider> <tool> --param value --bearer-token "$TOKEN"
```

Use generic invoke when a tool id is easier to pass around:

```bash
factory invoke <provider>.<tool> \
  --provider-params-json '{"bearer_token":"..."}' \
  --params-json '{"message":"hello"}'
```

Provider parameters such as `--bearer-token`, `--api-key`, and `--base-url` are optional by design. Some deployments may route through an authenticated proxy and not require direct provider credentials.

## Output Contract

By default, every command prints only:

```text
SUCCESS
full logs at /absolute/path/to/log.json
```

or:

```text
FAILURE
full logs at /absolute/path/to/log.json
```

The full result, structured error, events, command metadata, and duration are in the JSON log file. Read that file when you need data.

This keeps chat context small. Do not paste full logs into the conversation unless the user asks or the failure requires debugging.

## Debug Mode

Use `--debug` when the full response must be visible immediately:

```bash
factory --debug discover long google send-email
factory --debug google send-email --to user@example.com --subject Hi --body Hello
```

In debug mode:

- success JSON goes to stdout
- error JSON goes to stderr
- invocation logs are still written

## Failure Handling

On `FAILURE`:

1. Read the log path from stdout.
2. Inspect the JSON log.
3. Check the structured error:
   - `code`
   - `message`
   - `retryable`
   - `provider_status`
   - `details`
4. Retry only if `retryable` is true or the failure was caused by missing/incorrect parameters you can fix.

Never guess around destructive failures. Ask the user before retrying a command that may create, update, send, delete, charge, invite, or publish anything.

## Context-Safe Pattern

Use this production pattern:

1. `factory search "<intent>"`
2. Read the log and pick the highest relevant provider/tool.
3. `factory discover short <provider> [tool]`
4. If still relevant, `factory discover long <provider> <tool>`
5. Construct exact params from the long schema.
6. Invoke once.
7. Read the invocation log.
8. Summarize only the relevant result fields to the user.

## What Not To Do

- Do not ask for a full provider list; use semantic search.
- Do not run `discover long` across many tools.
- Do not paste full invocation logs into context by default.
- Do not assume credentials are stored.
- Do not run destructive e2e or production operations without explicit user confirmation.
- Do not retry non-retryable provider errors blindly.

