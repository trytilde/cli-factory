# Send Email Tool Generation Guidance

Use Gmail `users.messages.send`. Build an RFC 5322 message and base64url encode
it as the `raw` field. Preserve dry-run behavior for tests and safety checks.
