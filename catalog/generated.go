package catalog

import "embed"

//go:embed tools.json embeddings.json embeddings.bin
var FS embed.FS
