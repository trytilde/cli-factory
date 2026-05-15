# Google Workspace Provider Generation Guidance

Expose high-level Gmail and Google Calendar workflows, not raw Google API CRUD.
Use durable OAuth refresh-token credentials for real e2e tests. Keep test writes
limited to bot@trytilde.ai, send Gmail only to itself, create Calendar events on
the configured test calendar, and clean up test-created resources.
