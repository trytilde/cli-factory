# Check Emails Tool Generation Guidance

Use Gmail `users.messages.list` followed by `users.messages.get` with metadata
format for readable summaries. Keep output concise and avoid returning full raw
message bodies by default.
