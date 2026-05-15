# Google Workspace Provider Generation Guidance

Keep tools workflow-oriented and narrow. Prefer Gmail and Calendar REST endpoints
that map to common agent tasks. Do not add broad mailbox mutation or event
deletion tools without explicit safety requirements and e2e cleanup plans.

Provider auth is a caller-supplied OAuth 2.0 bearer token. Do not store refresh
tokens, client secrets, or user credentials.
